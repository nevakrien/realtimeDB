# realtimeDB
trying my hand at extremly low latancy dbs for linux

zig test src/talk.zig --library c

# protocol
all strings are send as a uint32 of length folowed by their charchters. 

a comunication starts with the client sending a protocol code of 1 byte
curently we have
0x00 : subscribe (network)
0x01 : publish (network)

subscribe connections can subscribe or unsubscribe by using:
subscribe(0x00 + string)
unsubscribe(0x00 + string)

# benchmarking 
benchmarking 

10 pubs each 1000 messages to a 100 subs

redis

Average Latency: 0.3566 ms
90th Percentile Latency: 0.5180 ms
95th Percentile Latency: 0.5974 ms
99th Percentile Latency: 0.8230 ms
99.9th Percentile Latency: 1.6506 ms

keydb 

Average Latency: 0.2676 ms
90th Percentile Latency: 0.4381 ms
95th Percentile Latency: 0.5215 ms
99th Percentile Latency: 0.7186 ms
99.9th Percentile Latency: 1.1327 ms


### inspired? 
https://stackoverflow.com/questions/26319304/redis-of-channels-degrading-latency-how-to-prevent-degradation