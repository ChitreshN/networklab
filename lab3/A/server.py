import asyncio
import struct
import time

MAGIC = 0xC461
VERSION = 1
HELLO = 0
DATA = 1
ALIVE = 2
GOODBYE = 3
HEADER_SIZE = 20

def unpack_header(data):
    return struct.unpack("!HBBIIQ", data[:HEADER_SIZE])

def pack_header(magic, version, command, seq, session_id, logical_clock):
    return struct.pack("!HBBIIQ", magic, version, command, seq, session_id, logical_clock)

class UAPServer:
    def __init__(self, port):
        self.port = port
        self.sessions = {}
        self.transport = None

    def process_message(self, data, addr):
        header = unpack_header(data)
        magic, version, command, seq, session_id, logical_clock = header

        if magic != MAGIC or version != VERSION:
            return 

        if session_id not in self.sessions:
            if command == HELLO:
                print(f"{hex(session_id)} [0] Session created")
                self.sessions[session_id] = {'seq': 0, 'addr': addr, 'clock': logical_clock}
                self.send_response(HELLO, session_id, 0, addr)
            else:
                return  

        session = self.sessions[session_id]
        next_seq = session['seq'] + 1
        if seq < session['seq']:
            print(f"{hex(session_id)} Duplicate packet")
        elif seq > next_seq:
            for lost_seq in range(session['seq'] + 1, seq):
                print(f"{hex(session_id)} Lost packet {lost_seq}")
            session['seq'] = seq
        else:
            session['seq'] = seq

        if command == DATA:
            print(f"{hex(session_id)} [{seq}] {data[HEADER_SIZE:].decode('utf-8')}")
            self.send_response(ALIVE, session_id, seq, addr)
        elif command == GOODBYE:
            print(f"{hex(session_id)} [{seq}] GOODBYE from client.")
            print(f"{hex(session_id)} Session closed")
            self.send_response(GOODBYE,session_id,seq,addr)
            del self.sessions[session_id]

    def send_response(self, command, session_id, seq, addr):
        response = pack_header(MAGIC, VERSION, command, seq, session_id, int(time.time()))
        self.transport.sendto(response, addr)

    def connection_made(self, transport):
        self.transport = transport
        print(f"Waiting on port {self.port}...")

    def datagram_received(self, data, addr):
        self.process_message(data, addr)

    def connection_lost(self, exc):
        if exc:
            print(f"Connection lost with exception: {exc}")
        else:
            print("Connection closed.")

    async def read_stdin(self):
        while True:
            try:
                line = await asyncio.get_event_loop().run_in_executor(None, input)
                if line.strip() == 'q':
                    await self.shutdown()
            except EOFError:
                await self.shutdown()

    async def shutdown(self):
        for session_id, session in self.sessions.items():
            print(f"Sending GOODBYE to session {hex(session_id)}")
            self.send_response(GOODBYE, session_id, session['seq'], session['addr'])

        # Wait to ensure messages are sent
        await asyncio.sleep(0.5)

        # Close the transport
        if self.transport:
            self.transport.close()

        # Allow asyncio to stop after all tasks are completed
        await asyncio.sleep(0.1)

async def main(port):
    loop = asyncio.get_running_loop()

    # Create the server endpoint
    listen = await loop.create_datagram_endpoint(
        lambda: UAPServer(port),
        local_addr=('0.0.0.0', port)
    )

    transport, protocol = listen

    # Start reading from stdin
    asyncio.create_task(protocol.read_stdin())

    try:
        # Run the event loop until the server is stopped
        await asyncio.sleep(float('inf'))
    except asyncio.CancelledError:
        # Handle any cleanup if needed
        pass

if __name__ == '__main__':
    import sys
    port = int(sys.argv[1])
    try:
        asyncio.run(main(port))
    except KeyboardInterrupt:
        print("Server interrupted and shutting down.")
