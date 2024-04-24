#include "common.h"
#include <stdint.h>



// Example function to subscribe to a channel
void subscribe_to_channel(int sockfd, const char *channel_name) {
    char op_code = 0x00;  // Subscribe operation
    write(sockfd, &op_code, sizeof(op_code));
    send_string(sockfd, channel_name);
}

// Function to unsubscribe from a channel
void unsubscribe_from_channel(int sockfd, const char *channel_name) {
    char op_code = 0x01;  // Unsubscribe operation
    write(sockfd, &op_code, sizeof(op_code));
    send_string(sockfd, channel_name);
}

// Example function to publish a message
void publish_message(int sockfd, const char *channel, const char *message) {
    send_string(sockfd, channel);
    send_string(sockfd, message);
}


// Structure to hold server messages
typedef struct ServerMessage {
    uint8_t type;       // 0 for regular message, 1 for error message
    uint32_t channel_length;
    uint32_t content_length;
    char *channel;  // Channel name
    char *content;  // Message content
} ServerMessage;