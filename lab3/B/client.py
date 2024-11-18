import socket
import struct
import random
import sys

MAGIC = 0xC461
VERSION = 1
HELLO = 0
DATA = 1
ALIVE = 2
GOODBYE = 3
HEADER_SIZE = 16
TIMEOUT = 5

def pack_buffer(magic, version, command, seq_number, session_id, logical_clock, message):
    byte_message = message.encode('utf-8')
    return struct.pack('!HBBIIQ', magic, version, command, seq_number, session_id, logical_clock) + byte_message

def unpack_buffer(packet):
    header = struct.unpack('!HBBIIQ', packet[:20])
    message = packet[20:].decode('utf-8')
    
    return {
        'magic': header[0],
        'version': header[1],
        'command': header[2],
        'seq_number': header[3],
        'session_id': header[4],
        'clock': header[5]
    }, message

def main(host,port):
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    server_address = (host, port)

    session_id = random.randint(0, 2**32 - 1)
    seq = 1

    hello_packet = pack_buffer(MAGIC, VERSION, HELLO, 0, session_id, 1, "")
    sock.sendto(hello_packet, server_address)

    while True:
        message = input()
        message.strip()
        command = DATA
        if message == 'q':
            command = GOODBYE
        packet = pack_buffer(MAGIC, VERSION, command, seq, session_id, 1, message)
        sock.sendto(packet, server_address)
        seq += 1

        buffer, _ = sock.recvfrom(1024)
        header, _ = unpack_buffer(buffer)

        if header['command'] == ALIVE:
            print("alive")
        elif header['command'] == GOODBYE:
            print("server sent goodbye")
            sys.exit(0)

if __name__ == "__main__":
    port = int(sys.argv[2])
    host = sys.argv[1]
    main(host, port)

