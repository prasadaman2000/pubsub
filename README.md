# This is a simple pubsub server.

### You can do two things.

* Subscribe
* Publish

*What else do you need?*

Everything is passed either thru URL params or POST body.

* You want to subscribe? Send a GET to `http://serverip:port/subscribe?topic=<topicname>&peerIp=<ip to publish messages to>&peerPort<port to publish messages to>`

* You want to publish? Send a POST to `http://serverip:port/subscribe?topic=<topicname>` with whatever body you want to send. I don't care and neither does the server.

### If you use this in any sort of production setting, godspeed, I'm literally only publishing this so I don't do a jank little `scp` to my raspberry pi where this server is gonna run at home.