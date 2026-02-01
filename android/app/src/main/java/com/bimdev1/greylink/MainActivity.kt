package com.bimdev1.greylink

import android.content.Context
import android.net.ConnectivityManager
import android.net.LinkProperties
import android.net.Network
import android.net.NetworkRequest
import android.os.Bundle
import android.util.Log
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import java.io.File
import org.json.JSONArray
import org.json.JSONObject
import transport.Transport
import java.net.Inet4Address

class MainActivity : AppCompatActivity() {
    private val TAG = "GreyLinkPoC"
    private lateinit var connectivityManager: ConnectivityManager
    
    // Store properties for all active networks
    // Key: Network Handle (String or ID), Value: LinkProperties
    private val activeNetworks = HashMap<String, LinkProperties>()
    
    // Lock for thread safety
    private val lock = Any()

    private val networkCallback = object : ConnectivityManager.NetworkCallback() {
        override fun onLinkPropertiesChanged(network: Network, linkProperties: LinkProperties) {
            synchronized(lock) {
                // Save the latest properties
                activeNetworks[network.toString()] = linkProperties
                pushNetworkState()
            }
        }

        override fun onLost(network: Network) {
            synchronized(lock) {
                activeNetworks.remove(network.toString())
                pushNetworkState()
            }
        }
        
        override fun onAvailable(network: Network) {
             // onAvailable is called when a network comes up. 
             // We generally wait for onLinkPropertiesChanged to get the details,
             // but we can try to fetch them now just in case.
             val props = connectivityManager.getLinkProperties(network)
             if (props != null) {
                 synchronized(lock) {
                     activeNetworks[network.toString()] = props
                     pushNetworkState()
                 }
             }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        // Initialize Connectivity Manager
        connectivityManager = getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        
        // Register for all network updates
        val request = NetworkRequest.Builder().build()
        connectivityManager.registerNetworkCallback(request, networkCallback)

        val etAuthKey = findViewById<EditText>(R.id.etAuthKey)
        val btnConnect = findViewById<Button>(R.id.btnConnect)
        val tvStatus = findViewById<TextView>(R.id.tvStatus)
        
        // Push initial state immediately (if any networks are known)
        // Note: activeNetworks might be empty until callbacks fire, which happens async.
        // We can manually populate activeNetwork if needed, but callback is safer.

        btnConnect.setOnClickListener {
            val authKey = etAuthKey.text.toString()
            if (authKey.isNotEmpty()) {
                startTransport(authKey, tvStatus)
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        try {
            connectivityManager.unregisterNetworkCallback(networkCallback)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to unregister callback", e)
        }
    }

    private fun pushNetworkState() {
        try {
            // Find the default (active) network's interface name
            val defaultNetwork = connectivityManager.activeNetwork
            var defaultIfName = ""
            
            if (defaultNetwork != null) {
                // If we have properties for the default network, use its interface name.
                // We trust our map, or we can fetch fresh.
                // Fetching fresh ensures we match the map's logic if we keyed by ID.
                val props = connectivityManager.getLinkProperties(defaultNetwork)
                if (props != null) {
                    defaultIfName = props.interfaceName ?: ""
                }
            }

            // Build JSON
            val root = JSONObject()
            val ifaceList = JSONArray()
            
            for ((_, props) in activeNetworks) {
                val name = props.interfaceName ?: continue
                
                // Fix for API < 29: LinkProperties.mtu is not available.
                // Use java.net.NetworkInterface instead.
                var mtu = 1500 // Default fallback
                try {
                    val netIntf = java.net.NetworkInterface.getByName(name)
                    if (netIntf != null) {
                        mtu = netIntf.mtu
                    }
                } catch (e: Exception) {
                    Log.w(TAG, "Failed to get MTU for $name", e)
                }
                
                val ifObj = JSONObject()
                ifObj.put("Name", name)
                ifObj.put("MTU", mtu)
                
                val addrList = JSONArray()
                for (la in props.linkAddresses) {
                    // Format: "IP/Prefix"
                    // InetAddress.hostAddress gives standard string
                    val addrStr = la.address.hostAddress
                    // Check for IPv6 scope ID (percent sign) and maybe strip it for CIDR parsing if needed?
                    // Go's net.ParseCIDR handles standard notation. 
                    // Android might return "fe80::1%wlan0".
                    // We should probably keep it clean.
                    val cleanAddr = if (addrStr != null && addrStr.contains("%")) {
                        addrStr.substringBefore("%")
                    } else {
                        addrStr
                    }
                    
                    if (cleanAddr != null) {
                        addrList.put("$cleanAddr/${la.prefixLength}")
                    }
                }
                ifObj.put("Addrs", addrList)
                ifaceList.put(ifObj)
            }
            
            root.put("Interfaces", ifaceList)
            root.put("DefaultInterface", defaultIfName)
            
            // Send to Go
            val len = ifaceList.length()
            Log.d(TAG, "Pushing network state: Default=$defaultIfName, Count=$len")
            Transport.updateNetworkState(root.toString())
            
        } catch (e: Exception) {
            Log.e(TAG, "Error pushing network state", e)
        }
    }

    private fun startTransport(authKey: String, tvStatus: TextView) {
        val stateDir = File(filesDir, "state").absolutePath
        
        Thread {
            try {
                runOnUiThread { tvStatus.text = "Status: Starting..." }
                
                // Call Go library
                // It now returns the Tailscale IP on success
                val ip = Transport.start(authKey, stateDir)
                
                runOnUiThread { tvStatus.text = "Status: Connected: $ip" }
                
                Log.d(TAG, "Transport started successfully: $ip")
                
            } catch (e: Exception) {
                Log.e(TAG, "Error starting transport", e)
                runOnUiThread { tvStatus.text = "Status: Error: ${e.message}" }
            }
        }.start()
    }
}
