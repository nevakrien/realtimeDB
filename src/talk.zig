//this is a dynamic disspatch implementation of the protocol
//all sockets are assumed to be nonblocking
const std = @import("std");
const os=std.os.linux;
const basics = @import("basics.zig");

const c = @cImport({
    @cInclude("sys/ioctl.h"); // Include the C header that contains FIONREAD
    @cInclude("unistd.h"); //read
    //@cInclude("errno.h");
});


const ProtocolType = enum { network_sub, network_pub,invalid };
const PubType = enum { network_pub };
const SubType = enum { network_sub };


// Define protocol-specific socket types
const SubNetworkSocket = struct  {
    fd: i32,
};

const PubNetworkSocket = struct  {
    fd: i32,
};

const AllSockets = union(ProtocolType) {
    network_sub: SubNetworkSocket,
    network_pub: PubNetworkSocket,
    invalid: u8,
};

const PubSockets = union(PubType){
	network_pub: PubNetworkSocket,
};

const SubSockets = union(SubType){
	network_sub: SubNetworkSocket,
};


const ReadErrors=error{SocketClosed,WouldBlock,ReadFailed,};

pub fn readByte(socket_fd: i32) ReadErrors!u8{
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
    return buffer[0];
}

fn printErrno() void{
    std.debug.print("\nerror {}",.{std.c._errno().*});
}

//reads the size of the string making a buffer.
pub fn readString(socket_fd: i32,buffer: *?[]u8 , allocFn: basics.AllocFn,comptime networkByteOrder: bool) ( ReadErrors || basics.MemoryError || error{OsError})![]u8{
    var len: u32  = undefined;
    if(buffer.*==null){
        var available: usize = 0;
        if(c.ioctl(socket_fd, c.FIONREAD, &available)<0){
            std.debug.print("\nerror with FIONREAD\n", .{});
            return error.OsError;
        }

        if (available < @sizeOf(u32)) return error.WouldBlock;
        
        
        const readbytes=c.read(socket_fd, @ptrCast(&len) ,@sizeOf(u32));
        if( readbytes != @sizeOf(u32)){
            printErrno();
            //std.debug.print("modified val bytes: {}",.{lenptr[9]});
            std.debug.print("\nExpected bytes: {}, Actual bytes read: {} and as a negative: {}\n", .{@sizeOf(u32), readbytes,0-%readbytes});
           // std.debug.print("\nread size did not match what the OS told us it would be\n", .{});
            return error.OsError;
        }

        if(networkByteOrder){
            len=std.mem.bigToNative(u32, len);
        }

        //std.debug.print("length is:{}\n",.{len});

        buffer.* = try allocFn(len);
    }
    else{
        len=@intCast(buffer.*.?.len);
    }

    const readbytes = os.read(socket_fd, @ptrCast(&len),len);
    return buffer.*.?[readbytes..];
}



test "readString functionality and error handling" {
    const alloc=basics.general_alloc;


    const tmp_file_path = "test_temp_file.tmp";

    // Create and write to a temporary file to simulate socket data
    var tmp_file = try std.fs.cwd().createFile(tmp_file_path, .{.read = true,});
    //defer tmp_file.close();
    //defer std.fs.cwd().deleteFile(tmp_file_path);

    // Write sample data in network byte order if needed
    const sample_data = "Hello, Zig!";
    const len: u32 = @intCast(sample_data.len);
    const lenBytes: [4]u8 = basics.u32ToBigEndianBytes(len);


    try tmp_file.writeAll(&lenBytes); // Assume network order for test
    try tmp_file.writeAll(sample_data);

    // Rewind file to simulate reading from start
    try tmp_file.seekTo(0);

    // Test reading string from file
    var buffer: ?[]u8 = null;
    const result = try readString(tmp_file.handle, &buffer, alloc, true);
    try std.testing.expectEqual(result.len,0);
    try std.testing.expectEqualStrings(sample_data, buffer.?);

    // Test handling insufficient data causing WouldBlock error
    try basics.truncateFile(tmp_file.handle,0);
    try tmp_file.seekTo(0);

    try tmp_file.writeAll(lenBytes[0..2]); // Write only part of the length
    try tmp_file.seekTo(0);
    buffer = null; // Reset buffer
    const readResult = readString(tmp_file.handle, &buffer, alloc, true);
    //const error = readResult catch |err| err;
    try std.testing.expectEqual(readResult, error.WouldBlock); // Expect WouldBlock error
}


pub fn establishSocket(socket_fd: i32) ReadErrors!AllSockets {
    const code=try readByte(socket_fd);

    // Return the appropriate socket type based on the protocol
    switch (code) {
        0x00 => return AllSockets{.network_sub= SubNetworkSocket{ .fd = socket_fd }},
        0x01 => return AllSockets{.network_pub= PubNetworkSocket{ .fd = socket_fd }},
        else => return AllSockets{.invalid=code},
    }
}

test "full errorset establishSocket" {
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
		else => return error.InvalidProtocolCode,
	}
}


const SubCodes = enum(u8) {
    subscribe = 0,
    unSubscribe = 1,
};


pub fn readSubNetwork(socket :SubNetworkSocket) (ReadErrors || error{InvalidActionCode})!void{
    const code=try readByte(socket.fd);
    const enum_info = @typeInfo(SubCodes).Enum;

    if (!enum_info.is_exhaustive) {
        if (std.math.cast(enum_info.tag_type, code)) |tag| {
            const validCode = @as(SubCodes, @enumFromInt(tag));
            switch(validCode){
                .subscribe=>{},
                .unsubscribe=>{},
            }
            return;
        }
        return error.InvalidActionCode;
    }
}

test "full errorset readSubNetwork" {
    //std.debug.print("\nenter a protocol code???...\n",.{});

    const r = readSubNetwork(SubNetworkSocket{.fd=-1})  catch |err| switch (err) {
            error.WouldBlock => {std.debug.panic("Panicked at Error: {any}",.{err});},
            error.ReadFailed => {std.debug.panic("Panicked at Error: {any}",.{err});},
            error.SocketClosed => {std.debug.panic("Panicked at Error: {any}",.{err});},
            error.InvalidActionCode => {std.debug.panic("Panicked at Error: {any}",.{err});},
         };
        
    std.debug.print("\nsocket type: {any}\n", .{r});
}

pub fn readPub(socket: PubSockets) !void {
	switch(socket){
		.network_pub => readPubNetwork(socket),
		else => return std.os.err.InvalidData,
	}
}

fn readPubNetwork(_ :PubNetworkSocket) void{

}