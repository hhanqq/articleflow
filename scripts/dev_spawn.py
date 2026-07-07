#!/usr/bin/env python3
import argparse
import subprocess
from pathlib import Path


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--workdir", required=True)
    parser.add_argument("--log", required=True)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()

    command_parts = args.command
    if command_parts and command_parts[0] == "--":
        command_parts = command_parts[1:]
    command = " ".join(command_parts).strip()
    if not command:
        raise SystemExit("command is required")

    Path(args.log).parent.mkdir(parents=True, exist_ok=True)
    log_file = open(args.log, "wb", buffering=0)
    process = subprocess.Popen(
        ["bash", "-lc", command],
        cwd=args.workdir,
        stdout=log_file,
        stderr=subprocess.STDOUT,
        start_new_session=True,
    )
    print(process.pid)


if __name__ == "__main__":
    main()
