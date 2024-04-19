#include <stdio.h>
#include <stdlib.h>
#include <sys/epoll.h>
#include <unistd.h>
#include <string.h>
#include <time.h>

#define ITERATIONS 10000000 // 10 million

int main() {
    int pipefd[2];
    char buf[30];
    struct epoll_event ev, events[1];
    int epollfd, nfds;
    struct timespec start, end;
    long long total_ns = 0;

    // Create a pipe
    if (pipe(pipefd) == -1) {
        perror("pipe");
        exit(EXIT_FAILURE);
    }

    // Create epoll instance
    epollfd = epoll_create1(0);
    if (epollfd == -1) {
        perror("epoll_create1");
        exit(EXIT_FAILURE);
    }

    // Add the read end of the pipe to the interest list
    ev.events = EPOLLIN;
    ev.data.fd = pipefd[0];
    if (epoll_ctl(epollfd, EPOLL_CTL_ADD, pipefd[0], &ev) == -1) {
        perror("epoll_ctl: pipefd[0]");
        exit(EXIT_FAILURE);
    }

    for (int i = 0; i < ITERATIONS; i++) {
        // Write something to the pipe
        write(pipefd[1], "Hello", 5);

        // Start timing
        clock_gettime(CLOCK_PROCESS_CPUTIME_ID, &start);

        // Wait for events
        nfds = epoll_wait(epollfd, events, 1, -1);
        if (nfds == -1) {
            perror("epoll_wait");
            exit(EXIT_FAILURE);
        }

        // Stop timing
        clock_gettime(CLOCK_PROCESS_CPUTIME_ID, &end);

        // Calculate the elapsed time in nanoseconds and accumulate
        total_ns += (end.tv_sec - start.tv_sec) * 1000000000LL + (end.tv_nsec - start.tv_nsec);

        // Clear the pipe
        read(pipefd[0], buf, 5);
    }

    // Calculate average time in nanoseconds
    double average_ns = total_ns / (double)ITERATIONS;
    printf("Average elapsed time: %.3f ns\n", average_ns);

    // Clean up
    close(pipefd[0]);
    close(pipefd[1]);
    close(epollfd);
    return 0;
}
