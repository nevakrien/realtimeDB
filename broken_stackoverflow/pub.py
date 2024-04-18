import redis
import datetime
import time
import json
import sys

import threading
import gevent
from gevent import monkey
monkey.patch_all()

def main(chan):
    redis_host = '10.235.13.29'
    r = redis.client.StrictRedis(host=redis_host, port=6379)
    while True:
        def getpkg():
            package = {'time': time.time(),
                        'signature' : 'content'
                      }

            return package

        #test 2: complex data
        now = json.dumps(getpkg())

        # send it
        r.publish(chan, now)
        print('Sending {0}'.format(now))
        #print 'data type is %s' % type(now)
        time.sleep(1)

def zerg_rush(n):
    for x in range(n):
        t = threading.Thread(target=main, args=(x,))
        t.setDaemon(True)
        t.start()

if __name__ == '__main__':
    num_of_chan = 10
    zerg_rush(num_of_chan)
    cnt = 0
    stop_cnt = 21
    while True:
        print('Waiting')
        cnt += 1
        if cnt == stop_cnt:
            sys.exit(0)
        time.sleep(30)