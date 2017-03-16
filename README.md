# About
A bidirectional HTTP proxy providing an HTTP communication pathway between `osCore` and `HCP`.

# Setup
### Dependencies:
	
- Go (tested on 1.7)

### Compiling	
- Test: `go test ./..`  
- Compile: `go build cisco.com/comm`
- Run: `./comm --help`

# Usage

## Routing
**_NOTE this section is under revision as the routing mechanism is changing, take everything below with a grain of salt_**

Requests sent to the proxy will be routed dynamically based on a routing key.

- First, start the server with API on port 3500 and socket server on 3501

		server ~ $ ./comm --mode server -apiport 3500 -p 3501
- Start a client connecting to remote TCP 3501 and serving api on 3499

		client ~ $ ./comm --mode client -p 3501 -apiport 3499
- Now that they're connected, we can ask the server for a list of clients it is currently connected to:
		
		server ~ $ curl localhost:3500/connections
		{
		  "connections": [
    		{
      			"id": 1,
      			"remote": {
        			"IP": "127.0.0.1",
        			"Port": 59310,
        			"Zone": ""
      			}
   	 		}]
		}
- So the server has a connection with ID 1. Now on the client:

		client ~ $ curl localhost:3499/connections | jq
		{
		  "connections": [
		    {
		      "id": 0,
		      "remote": {
		        "IP": "127.0.0.1",
		        "Port": 3501,
		        "Zone": ""
		      }
		    }]
		}
- The client has connection ID 0 with the server. _Note that since clients can only connect to one server, the ID will always either be 0 or empty (if not connected)._

Then to proxy an HTTP call, send your HTTP requests to the _**LAN**_ server (this is not the same as the HTTP server. They listen on different ports and can be user-configured or auto configuring) prepended with the routing key. The destination node within the connected subnet should be specified in the `Host` header. For example, the following request:
 
	$ curl -H "Host: server1.pepsi.com" -XPUT local.proxy.server/1/foo -d PONG
Will be sent to the client given by connection `1` and proxied to `server1.pepsi.com` within that subnet as:

	$ curl -H "Host: server1.pepsi.com" -XPUT server1.pepsi.com/foo -d PONG
	
_Note that since the client can only be connected to a single server, it is not necessary to include the connection ID in the URL._


## Examples

### Proxy Server
This example demonstrates using this project as an HTTP proxy. This is probably the most common use case.

1. First, start a server
		
		server ~ $ ./comm
		INFO[0000] [socket.server] WAN server listening on [::]:57445
2. Now, connect a client

		client ~ $ ./comm --mode client -s localhost -p 57445
		INFO[0000] [main] Starting in CLIENT mode. Connecting to localhost:57445
		2017/03/09 16:08:41 Listening for LAN connections on TCP [::]:57455
		
3. Now, start an endpoint you wish to connect to through the proxy. Currently, the destination is hard-coded to connect to `localhost:8080`. This will change in the future to allow for dynamic routing. For now, lets start up a Python web server on the server:

		server ~ $ python -m SimpleHTTPServer 8080

4. Finally, send some data through the client-side of the proxy destined for this web server (note the vice versa case works just the same). This request will be proxied over the TCP channel to the server (in this example, both running on the same machine) and finally, to the Python web server. To the destination, the connection will appear to originate from the HTTP proxy server:

		client ~ $ curl localhost:57455
		<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 3.2 Final//EN"><html>
		<title>Directory listing for /</title>
		<body>
		<h2>Directory listing for /</h2>
		<hr>
		<ul>
		<li><a href=".DS_Store">.DS_Store</a>
		<li><a href=".idea/">.idea/</a>
		<li><a href="comm">comm</a>
		<li><a href="communication.iml">communication.iml</a>
		<li><a href="Makefile">Makefile</a>
		<li><a href="pkg/">pkg/</a>
		<li><a href="README.md">README.md</a>
		<li><a href="resources/">resources/</a>
		<li><a href="src/">src/</a>
		</ul>
		<hr>
		</body>
		</html>
		

**Considerations**: This code is still in very early stages of development. As such, bugs are to be expected.

		
### REPL
You can start an interactive REPL. Here, any string you write to the server will be echoed back converted into uppercase characters. This is mostly just for pretty demos or to sanity check the setup is working. It may go away in the future.


- Start the server

		$ ./comm --mode server
		2017/02/28 17:56:31  [x] Starting in SERVER mode. TCP bind 3500
- Start the client in interactive mode

		$ ./comm --mode client
		$ go build cisco.com/comm && ./comm -mode client
		...
		2017/02/28 18:00:49  [CLIENT] Connected to server
 		  [x] Enter message to send (Ctrl-C to exit) => it works
		...
		######################################################################
 		# Message RECV
	 	# Type: CONTROL Message . Length: 8 
 		# Payload: IT WORKS <nil> 
 		######################################################################

 		  [x] Enter message to send (Ctrl-C to exit) => 

# Architecture
**_It is not necessary to understand the information below to use this package, it is provided soley for documentation purposes_**

Every message consists of a 14byte header and variable-length payload:

```
0        5      6               14                22
+--------+------+----------------+----------------+
| Magic  | Type | Payload Length |     Sequence   |
|  (5)   |  (1) |      (64)      |       (64)     |
+--------+------+----------------+----------------+
|                Payload (Variable)               |
+-------------------------------------------------+
```

- `Magic` is a constant magic number.
- `Type` is a semantic type for the message. This is an arbitrary 8 bit number. Its meaning is left up to the consumers of this package. Think of this as an extension of Websockets' 1 bit "binary" vs "non-binary" type field. It is not necessary for the response message to be of the same type as the unsolicited message.
- `Payload Length` specifies the length, in bytes, of the payload. This does not include the header length. Make **sure** the length is correct. If it is too small, the next message will be discarded and the connection closed. If it is too large, you will end up reading into the next message which will most likely mean the subsequent message will be discarded and the connection closed.
- `Sequence` is an 8 byte request identifier.

