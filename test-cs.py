#!/usr/bin/env python3

import argparse
from socket import *
import time
import logging

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("server_addr", help="Server address")
    parser.add_argument("server_port", help="Server port", type=int)
    parser.add_argument("function_name", help="Function to be run")
    parser.add_argument("log_file", help="Log file name")
    args = parser.parse_args()

    logging.basicConfig(filename=args.log_file, filemode="a", format="%(message)s", level=logging.INFO)

    with socket(AF_INET, SOCK_STREAM) as sock:
        sock.connect((args.server_addr, args.server_port))
        payload = '{"hello": "world"}'
        message = f"""POST /run/{args.function_name} HTTP/1.1\r
Host:{args.server_addr}:{args.server_port}\r
Content-Type: application/x-www-form-urlencoded\r
Content-Length: {len(payload)}\r
\r
{payload}\r
\r\n"""

        # print(message)

        time_before = int(round(time.time() * 1000))
        sock.send(message.encode())
        resp = sock.recv(1024).decode()
        time_after = int(round(time.time() * 1000))

        time_diff = time_after - time_before
        logging.info(time_diff)
        print(time_diff)

        # print(resp)


if __name__ == '__main__':
    main()
