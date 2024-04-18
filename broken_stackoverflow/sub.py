import redis
import threading
import time
import json
import gevent
from gevent import monkey
monkey.patch_all()

def callback(ind):
    redis_host = '10.235.13.29'
    r = redis.client.StrictRedis(host=redis_host, port=6379)
    sub = r.pubsub()
    sub.subscribe(str(ind))
    start = False
    avg = 0
    tot = 0
    sum = 0
    while True:
        for m in sub.listen():
            if not start:
                start = True
                continue
            got_time = time.time()
            decoded = json.loads(m['data'])
            sent_time = float(decoded['time'])
            dur = got_time - sent_time
            tot += 1
            sum += dur
            avg = sum / tot

            #print decoded #'Recieved: {0}'.format(m['data'])
            print(decoded) #'Recieved: {0}'.format(m['data'])
            file_name = 'logs/sub_%s' % ind
            f = open(file_name, 'a')
            f.write('processing no. %s' % tot)
            f.write('it took %s' % dur)
            f.write('current avg: %s\n' % avg)
            f.close()
            print('wrote')

def zerg_rush(n):
    for x in range(n):
        t = threading.Thread(target=callback, args=(x,))
        t.setDaemon(True)
        t.start()

def main():
    num_of_chan = 10
    zerg_rush(num_of_chan)
    while True:
        #print 'Waiting'
        time.sleep(30)

if __name__ == '__main__':
    main()