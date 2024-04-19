#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/epoll.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <unistd.h>
#include <pthread.h>
#include <time.h>

#define PORT 8080

#define ITERATIONS 100000 // 100k

void *sender_thread(void *arg) {
    int sock = *(int *)arg;
    struct timespec send_time;

    //alow the first thread to exist
    usleep(100000);

    for(int i=0;i<ITERATIONS;i++){
        // Give time for epoll to be ready
        usleep(10);

        // Get current time and send it
        clock_gettime(CLOCK_PROCESS_CPUTIME_ID, &send_time);
        uint64_t time_ns = (uint64_t)send_time.tv_sec * 1000000000 + send_time.tv_nsec;
        write(sock, &time_ns, sizeof(uint64_t));
    }
    

    return NULL;
}

void *receiver_thread(void *arg) {
    int epollfd = *(int *)arg;
    struct epoll_event events;
    struct timespec recv_time;
    struct timespec read_time;
    uint64_t send_ns, recv_delay_ns , read_delay_ns;

    long long total_nano_wake=0;
    long long total_nano_read=0;

    for(int i=0;i<ITERATIONS;i++){
        // Wait for events
        int nfds = epoll_wait(epollfd, &events, 1, -1);
        if (nfds == -1) {
            perror("epoll_wait");
            exit(EXIT_FAILURE);
        }
        // Measure time immediately after waking
        clock_gettime(CLOCK_PROCESS_CPUTIME_ID, &recv_time);

        // Read data
        read(events.data.fd, &send_ns, sizeof(uint64_t));

        // Measure time after reading
        clock_gettime(CLOCK_PROCESS_CPUTIME_ID, &read_time);

        // Calculate delay
        uint64_t recv_cur_ns = (uint64_t)recv_time.tv_sec * 1000000000 + recv_time.tv_nsec;
        uint64_t read_cur_ns = (uint64_t)read_time.tv_sec * 1000000000 + read_time.tv_nsec;

        recv_delay_ns = recv_cur_ns - send_ns;
        read_delay_ns = read_cur_ns - send_ns;
        
        total_nano_wake+=recv_delay_ns;
        total_nano_read+=read_delay_ns;
    }
    long double avrg=total_nano_wake/ITERATIONS;
    printf("Avrage Delay Waking: %Lf ns\n", avrg);
    avrg=total_nano_read/ITERATIONS;
    printf("Avrage Delay Reading: %Lf ns\n", avrg);

    return NULL;
}

int main() {
    int sockfd[2], epollfd;
    struct epoll_event ev;

    // Create a socket pair
    if (socketpair(AF_UNIX, SOCK_STREAM, 0, sockfd) != 0) {
        perror("socketpair");
        exit(EXIT_FAILURE);
    }

    // Create epoll instance
    epollfd = epoll_create1(0);
    if (epollfd == -1) {
        perror("epoll_create1");
        exit(EXIT_FAILURE);
    }

    // Add one socket to epoll
    ev.events = EPOLLIN;
    ev.data.fd = sockfd[1];
    if (epoll_ctl(epollfd, EPOLL_CTL_ADD, sockfd[1], &ev) == -1) {
        perror("epoll_ctl");
        exit(EXIT_FAILURE);
    }

    pthread_t sender, receiver;
    pthread_create(&sender, NULL, sender_thread, &sockfd[0]);
    pthread_create(&receiver, NULL, receiver_thread, &epollfd);

    pthread_join(sender, NULL);
    pthread_join(receiver, NULL);

    close(sockfd[0]);
    close(sockfd[1]);
    close(epollfd);
    return 0;
}
