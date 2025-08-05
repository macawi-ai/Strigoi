#!/usr/bin/env python3
import pty
import os
import sys
import time

def main():
    # Create a pseudo-terminal
    master, slave = pty.openpty()
    
    # Fork a child process
    pid = os.fork()
    
    if pid == 0:  # Child process
        # Close master end
        os.close(master)
        
        # Make slave the controlling terminal
        os.setsid()
        os.dup2(slave, 0)  # stdin
        os.dup2(slave, 1)  # stdout
        os.dup2(slave, 2)  # stderr
        
        # Close the original slave fd
        os.close(slave)
        
        # Output some test data with secrets
        print("Starting PTY test...")
        print("API_KEY=sk-test-1234567890abcdef")
        print("DATABASE_PASSWORD=SecretPass123!")
        sys.stdout.flush()
        
        # Keep running for a bit
        for i in range(20):
            time.sleep(1)
            print(f"Running... {i+1}/20")
            if i == 10:
                print("TOKEN=Bearer-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")
                print("AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
            sys.stdout.flush()
        print("PTY test complete.")
        sys.exit(0)
        
    else:  # Parent process
        # Close slave end
        os.close(slave)
        
        print(f"Started PTY process with PID: {pid}")
        
        # Wait for child to finish
        os.waitpid(pid, 0)
        os.close(master)

if __name__ == "__main__":
    main()