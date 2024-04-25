//this is a dynamic disspatch implementation of the protocol
//all sockets are assumed to be nonblocking
const std = @import("std");
const os=std.os.linux;

const ProtocolType = enum { network_sub, network_pub,invalid };
const PubType = enum { network_pub };
const SubType = enum { network_sub };


// Define protocol-specific socket types
const NetworkSubSocket = struct  {
    fd: i32,

    // // Subscribe-specific methods
    // subscribe: fn (self: NetworkSubSocket, topic: []const u8) void,//anyerror!void,
    // unsubscribe: fn (self: NetworkSubSocket, topic: []const u8) void,//anyerror!void,
};

const NetworkPubSocket = struct  {
    fd: i32,

    // // Publish-specific methods
    // publish: fn (self: NetworkPubSocket, topic: []const u8, message: []const u8) void,//anyerror!void,
};

const AllSockets = union(ProtocolType) {
    network_sub: NetworkSubSocket,
    network_pub: NetworkPubSocket,
    invalid: u8,
};

const PubSockets = union(PubType){
	network_pub: NetworkPubSocket,
};

const SubSockets = union(SubType){
	network_sub: NetworkSubSocket,
};


pub fn establishSocket(socket_fd: i32) error{SocketClosed,WouldBlock,ReadFailed,}!AllSockets {
    var buffer: [1]u8 = undefined;  // Buffer to read one byte

    // Set the socket to non-blocking mode (pseudo code)
    // os.setsockopt(socket_fd, os.SO_NONBLOCK, 1) catch return error.FailedToSetNonBlocking;

    // Attempt to read one byte from the socket
    const bytes_read = os.read(socket_fd, buffer[0..],buffer.len);
    if (bytes_read == 0) {
        return error.SocketClosed;  // Handle unexpected end of file (socket closed)
    } else if (bytes_read == -1) {
        switch (os.errno()) {
            os.EAGAIN, os.EWOULDBLOCK => return error.WouldBlock,
            else => return error.ReadFailed,
        }
    }

    // Return the appropriate socket type based on the protocol
    switch (buffer[0]) {
        0x00 => return AllSockets{.network_sub= NetworkSubSocket{ .fd = socket_fd }},
        0x01 => return AllSockets{.network_pub= NetworkPubSocket{ .fd = socket_fd }},
        else => return AllSockets{.invalid=buffer[0]},
    }
}

test "full errorset" {
    //std.debug.print("\nenter a protocol code???...\n",.{});

    const r = establishSocket(-1)  catch |err| switch (err) {
            error.WouldBlock => {std.debug.panic("Panicked at Error: {any}",.{err});},
            error.ReadFailed => {std.debug.panic("Panicked at Error: {any}",.{err});},
            error.SocketClosed => {std.debug.panic("Panicked at Error: {any}",.{err});},
         };
        
    std.debug.print("\nsocket type: {any}\n", .{r});
}

pub fn readSub(socket: SubSockets) !void {
	switch(socket){
		.network_sub => readSubNetwork(socket),
		else => return std.os.err.InvalidData,
	}
}


const SubCodes = enum(u8) {
    subscribe = 0,
    unSubscribe = 1,
};


fn readSubNetwork(socket :NetworkSubSocket) !void{
    const code: [1]u8 = undefined;
    std.os.read(socket.fd,code);

    // swich(code[0]){
    //     subscribe=>
    // }
}

pub fn readPub(socket: PubSockets) !void {
	switch(socket){
		.network_pub => readPubNetwork(socket),
		else => return std.os.err.InvalidData,
	}
}

fn readPubNetwork(_ :NetworkPubSocket) void{

}