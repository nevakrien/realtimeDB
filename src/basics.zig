const std = @import("std");
pub const MemoryError=std.mem.Allocator.Error;
pub const AllocFn = fn ( n: usize) MemoryError![]u8;

var general_purpose_allocator = std.heap.GeneralPurposeAllocator(.{}){};
    //defer _ = general_purpose_allocator.deinit();
const gpa = general_purpose_allocator.allocator();

pub fn general_alloc(size: usize) error{OutOfMemory}![]u8 {
        return gpa.alloc(u8, size) catch error.OutOfMemory;
}

pub fn general_free(mem: ?[*]u8) void{
    gpa.free(mem);
}

// Function to convert a u32 to a byte array in big-endian format
pub fn u32ToBigEndianBytes(value: u32) [4]u8 {
    var bytes: [4]u8 = undefined; // Create an uninitialized array for the bytes
    bytes[0] = @intCast((value >> 24) & 0xFF);
    bytes[1] = @intCast((value >> 16) & 0xFF);
    bytes[2] = @intCast((value >> 8) & 0xFF);
    bytes[3] = @intCast(value & 0xFF);
    return bytes;
}

pub fn truncateFile(fd: i32, length: i64) !void {
    const result = std.os.linux.ftruncate(fd, length);
    if (result != 0) {
        //const errno = std.os.errno(result);
        //std.log.err("Error truncating file: {}", .{std.os.strerror(errno)});
        return error.FailedToTruncate;
    }
}