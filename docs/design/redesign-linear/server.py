#!/usr/bin/env python3
"""Serve the redesign-linear gallery + variations locally with no caching."""
import argparse
import http.server
import os
import socketserver
import sys


class Handler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header("Cache-Control", "no-store, no-cache, must-revalidate")
        self.send_header("Pragma", "no-cache")
        self.send_header("Expires", "0")
        super().end_headers()

    def log_message(self, format, *args):
        sys.stderr.write("%s - %s\n" % (self.address_string(), format % args))


def main():
    parser = argparse.ArgumentParser(description="Serve redesign-linear locally.")
    parser.add_argument("-p", "--port", type=int, default=8002)
    parser.add_argument("-b", "--bind", default="127.0.0.1")
    args = parser.parse_args()

    root = os.path.dirname(os.path.abspath(__file__))
    os.chdir(root)

    socketserver.TCPServer.allow_reuse_address = True
    with socketserver.TCPServer((args.bind, args.port), Handler) as httpd:
        url = f"http://{args.bind}:{args.port}/"
        print(f"Serving {root} at {url}")
        try:
            httpd.serve_forever()
        except KeyboardInterrupt:
            print("\nShutting down.")
            httpd.shutdown()


if __name__ == "__main__":
    main()
