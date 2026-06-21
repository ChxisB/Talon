const std = @import("std");

fn timestamp() i64 {
    var tv: std.c.timeval = undefined;
    _ = std.c.gettimeofday(&tv, null);
    return @as(i64, tv.sec);
}

fn microTimestamp() i64 {
    var tv: std.c.timeval = undefined;
    _ = std.c.gettimeofday(&tv, null);
    return @as(i64, tv.sec) * 1_000_000 + @as(i64, tv.usec);
}

const Mutex = struct {
    inner: std.atomic.Value(u32) = std.atomic.Value(u32).init(0),

    fn lock(m: *Mutex) void {
        while (m.inner.swap(1, .acquire) != 0) {
            while (m.inner.load(.monotonic) != 0) {}
        }
    }

    fn unlock(m: *Mutex) void {
        m.inner.store(0, .release);
    }
};

pub const LogLevel = enum(u8) {
    err = 0,
    warn = 1,
    info = 2,
    debug = 3,
};

var log_file: ?std.fs.File = null;
var log_mutex: Mutex = .{};
var initialized: bool = false;

/// Initialize the file logger with a timestamped filename (called automatically on first use)
fn ensureInit() void {
    if (initialized) return;

    const timestamp = timestamp();
    var filename_buf: [128]u8 = undefined;
    const filename = std.fmt.bufPrint(&filename_buf, "opentui_debug_{d}.log", .{timestamp}) catch return;

    const fd = std.posix.openat(std.posix.AT.FDCWD, filename, .{ .ACCMODE = .WRONLY, .CREAT = true, .TRUNC = true }, 0o644) catch return;
    log_file = std.io.File{ .handle = fd };

    // Log initialization
    const init_msg = std.fmt.bufPrint(&filename_buf, "=== Log initialized at timestamp {d} ===\n", .{timestamp}) catch return;
    _ = log_file.?.write(init_msg) catch return;
    log_file.?.sync() catch return;

    initialized = true;
}

/// Close the log file
pub fn deinit() void {
    log_mutex.lock();
    defer log_mutex.unlock();

    if (log_file) |file| {
        file.close();
        log_file = null;
        initialized = false;
    }
}

/// Log a message with level, file, line info and immediate flush
pub fn logMessage(level: LogLevel, comptime format: []const u8, args: anytype) void {
    log_mutex.lock();
    defer log_mutex.unlock();

    // Auto-initialize on first use
    if (!initialized) {
        ensureInit();
    }

    if (log_file == null) return;

    var buf: [8192]u8 = undefined;

    const level_str = switch (level) {
        .err => "ERROR",
        .warn => "WARN ",
        .info => "INFO ",
        .debug => "DEBUG",
    };

    const timestamp = microTimestamp();

    const msg = std.fmt.bufPrint(&buf, "[{d}] {s}: ", .{ timestamp, level_str }) catch return;
    _ = log_file.?.write(msg) catch return;

    const user_msg = std.fmt.bufPrint(&buf, format, args) catch return;
    _ = log_file.?.write(user_msg) catch return;
    _ = log_file.?.write("\n") catch return;

    // CRITICAL: Flush immediately so logs are on disk even if we crash
    log_file.?.sync() catch return;
}

pub fn err(comptime format: []const u8, args: anytype) void {
    logMessage(.err, format, args);
}

pub fn warn(comptime format: []const u8, args: anytype) void {
    logMessage(.warn, format, args);
}

pub fn info(comptime format: []const u8, args: anytype) void {
    logMessage(.info, format, args);
}

pub fn debug(comptime format: []const u8, args: anytype) void {
    logMessage(.debug, format, args);
}
