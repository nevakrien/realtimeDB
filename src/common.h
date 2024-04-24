#ifndef COMMON_H
#define COMMON_H

#include <stdint.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <arpa/inet.h>  // Include for htonl, ntohl functions

typedef struct StringArray{
    uint32_t size;
    char data[];
} StringArray;

StringArray* read_string(int fd) {
    uint32_t len;
    int n = read(fd, &len, sizeof(len));
    if (n <= 0) return NULL;
    len = ntohl(len); // Network to host byte order

    StringArray* out = malloc(len + sizeof(StringArray));
    if(!out){
        return NULL;
    }

    while(len){
        n = read(fd, out->data, len);
        if (n <0){
            free(out);
            return NULL;
        } 
        len-=n;

    }
    return out;
}
int send_string(int sockfd, const StringArray *str) {
    uint32_t network_len = htonl(str->size);  // Convert length to network byte order
    if(write(sockfd, &network_len, sizeof(network_len))) return 1;   // Send the length
    if(write(sockfd, str->data, str->size)) return 1;    // Send the string
    return 0;
}

// oxilary
int read_c_string(int fd, char **out) {
    uint32_t len;
    int n = read(fd, &len, sizeof(len));
    if (n <= 0) perror("opsy"); return n;
    len = ntohl(len); // Network to host byte order

    *out = malloc(len + 1);
    n = read(fd, *out, len);
    (*out)[len] = '\0';  // Null terminate the string
    return n;
}

int send_c_string(int sockfd, const char *str) {
    uint32_t len = strlen(str);
    uint32_t network_len=htonl(len);  // Convert length to network byte order
    if(write(sockfd, &network_len, sizeof(network_len))) return 1;   // Send the length
    if(write(sockfd, str,len)) return 1;    // Send the string
    return 0;
}

#endif //COMMON_H