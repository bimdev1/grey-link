package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"sync"
	"time"

	"tailscale.com/net/netmon"
	"tailscale.com/net/netutil"
	"tailscale.com/tsnet"
)

var (
	srv *tsnet.Server
	mu  sync.Mutex
)

// data pushed from Android
type AndroidState struct {
	Interfaces       []AndroidInterface
	DefaultInterface string
}

type AndroidInterface struct {
	Name  string
	MTU   int
	Addrs []string // CIDR strings e.g. "192.168.1.5/24"
}

var (
	stateMu   sync.RWMutex
	lastState AndroidState
)

func init() {
	// 1. Hook into netmon to provide the full interface list
	netmon.RegisterInterfaceGetter(getAndroidInterfaces)

	// 2. Hook into netutil to provide the default interface
	netutil.GetDefaultInterface = getAndroidDefaultInterface
}

// UpdateNetworkState is called by the Android app via gomobile
// whenever ConnectivityManager detects a change.
func UpdateNetworkState(jsonStr string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("transport: PANIC in UpdateNetworkState: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	var newState AndroidState
	if err := json.Unmarshal([]byte(jsonStr), &newState); err != nil {
		log.Printf("transport: failed to unmarshal android state: %v", err)
		return err
	}

	log.Printf("transport: received network update: def=%s, ifaces=%d", newState.DefaultInterface, len(newState.Interfaces))

	stateMu.Lock()
	lastState = newState
	stateMu.Unlock()

	// Notify netmon about the route change immediately
	if newState.DefaultInterface != "" {
		netmon.UpdateLastKnownDefaultRouteInterface(newState.DefaultInterface)
	}

	return nil
}

func getAndroidInterfaces() ([]netmon.Interface, error) {
	stateMu.RLock()
	defer stateMu.RUnlock()

	if len(lastState.Interfaces) == 0 {
		return []netmon.Interface{
			{
				Interface: &net.Interface{
					Index: 1,
					MTU:   1500,
					Name:  "dummy0",
					Flags: net.FlagUp | net.FlagLoopback | net.FlagMulticast,
				},
				AltAddrs: []net.Addr{
					&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)},
				},
			},
		}, nil
	}

	var ret []netmon.Interface
	for i, aIf := range lastState.Interfaces {
		// Construct net.Interface
		nIf := &net.Interface{
			Index: i + 1,
			MTU:   aIf.MTU,
			Name:  aIf.Name,
			Flags: net.FlagUp | net.FlagMulticast | net.FlagBroadcast,
		}
		if aIf.Name == "lo" || aIf.Name == "wlan0" {
			nIf.Flags |= net.FlagMulticast
		}

		// Parse Addresses
		var altAddrs []net.Addr
		for _, cidr := range aIf.Addrs {
			ip, ipnet, err := net.ParseCIDR(cidr)
			if err == nil {
				fullAddr := &net.IPNet{
					IP:   ip,
					Mask: ipnet.Mask,
				}
				altAddrs = append(altAddrs, fullAddr)
			}
		}

		ret = append(ret, netmon.Interface{
			Interface: nIf,
			AltAddrs:  altAddrs,
		})
	}
	return ret, nil
}

func getAndroidDefaultInterface() (string, netip.Addr, error) {
	stateMu.RLock()
	defer stateMu.RUnlock()

	defName := lastState.DefaultInterface
	if defName == "" {
		// Don't return error during init/early phase, return dummy loopback
		// so netutil stops complaining regarding default route lookup.
		// netutil.DefaultInterfacePortable calls this.
		// If we return valid loopback, it might be happy.
		return "dummy0", netip.AddrFrom4([4]byte{127, 0, 0, 1}), nil
	}

	// Find the IP of the default interface
	for _, iface := range lastState.Interfaces {
		if iface.Name == defName {
			for _, cidr := range iface.Addrs {
				// Return the first IPv4 address preferably
				ip, _, err := net.ParseCIDR(cidr)
				if err == nil {
					if ip4 := ip.To4(); ip4 != nil {
						addr, _ := netip.AddrFromSlice(ip4)
						return defName, addr, nil
					}
				}
			}
			// Fallback to IPv6 if no v4
			for _, cidr := range iface.Addrs {
				ip, _, err := net.ParseCIDR(cidr)
				if err == nil {
					addr, _ := netip.AddrFromSlice(ip)
					return defName, addr, nil
				}
			}
		}
	}

	return "", netip.Addr{}, fmt.Errorf("default interface %q found but has no IP", defName)
}

// Start initializes the tsnet server and returns the Tailscale IP
func Start(authKey string, stateDir string) (ip string, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("transport: PANIC in Start: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	os.Setenv("TS_DEBUG_ALWAYS_USE_DERP", "true")
	os.Setenv("TS_DEBUG_SKIP_BIND_DETECT", "true")

	// Fix for "panic: no safe place found to store log state"
	// Android doesn't set XDG_CACHE_HOME, so filch (logtail) fails to find a cache dir.
	cacheDir := stateDir + "/cache"
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %v", err)
	}
	os.Setenv("XDG_CACHE_HOME", cacheDir)
	// Also set HOME just in case other things need it
	os.Setenv("HOME", stateDir)

	mu.Lock()
	defer mu.Unlock()

	if srv != nil {
		return "", fmt.Errorf("server already started")
	}

	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create state dir: %v", err)
	}

	srv = &tsnet.Server{
		Hostname: "grey-link-android",
		AuthKey:  authKey,
		Dir:      stateDir,
		Logf: func(format string, args ...any) {
			log.Printf("TSNET: "+format, args...)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	lc, err := srv.LocalClient()
	if err != nil {
		return "", fmt.Errorf("failed to get local client: %v", err)
	}

	// Wait for IP address to be assigned (poll for up to 5 seconds)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st, err := lc.Status(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get status: %v", err)
		}
		if len(st.TailscaleIPs) > 0 {
			log.Printf("Grey-Link transport started. Tailscale IP: %v", st.TailscaleIPs)
			return st.TailscaleIPs[0].String(), nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Retrieve status one last time to report state
	st, err := lc.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get status: %v", err)
	}

	log.Printf("Grey-Link transport started but IP is not yet ready. Status: %v", st.BackendState)
	return "connected-waiting-for-ip", nil
}

func TestConnection(addr string) (string, error) {
	mu.Lock()
	s := srv
	mu.Unlock()

	if s == nil {
		return "", fmt.Errorf("server not started")
	}

	conn, err := s.Dial(context.Background(), "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("dial failed: %v", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "PING\n"); err != nil {
		return "", fmt.Errorf("write failed: %v", err)
	}

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		return "", fmt.Errorf("read failed: %v", err)
	}

	return string(buf[:n]), nil
}

func Stop() error {
	mu.Lock()
	defer mu.Unlock()
	if srv != nil {
		err := srv.Close()
		srv = nil
		return err
	}
	return nil
}
