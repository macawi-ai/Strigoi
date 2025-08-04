#!/usr/bin/env python3
"""
Vulnerable application for testing Strigoi Center module.
This app deliberately exposes credentials in various ways.

WARNING: This is for testing only! Never run in production!
"""
import json
import time
import sys
import os

def test_json_credentials():
    """Expose credentials in JSON format."""
    config = {
        "database": {
            "host": "prod.db.internal",
            "user": "admin",
            "password": "SuperSecret123!",
            "port": 3306
        },
        "api_keys": {
            "openai": "sk-1234567890abcdefghijklmnopqrstuv",
            "github": "ghp_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
            "aws_key": "AKIAIOSFODNN7EXAMPLE"
        }
    }
    print("Loading configuration...")
    print(json.dumps(config, indent=2))
    sys.stdout.flush()

def test_sql_passwords():
    """Expose passwords in SQL queries."""
    queries = [
        "CREATE USER 'newuser' IDENTIFIED BY 'P@ssw0rd123';",
        "mysql://dbuser:MyDBP@ss123@localhost:3306/production",
        "UPDATE users SET password='NewSecretPass' WHERE id=1;",
        "GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' IDENTIFIED BY 'RootPass!';",
    ]
    
    for query in queries:
        print(f"Executing SQL: {query}")
        sys.stdout.flush()
        time.sleep(0.5)

def test_environment_exposure():
    """Expose environment variables."""
    # Set some fake environment variables
    os.environ['AWS_SECRET_ACCESS_KEY'] = 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY'
    os.environ['GITHUB_TOKEN'] = 'ghp_ThisIsAFakeGitHubTokenForTesting'
    os.environ['DATABASE_URL'] = 'postgres://user:password@host:5432/dbname'
    
    print("Environment check:")
    for key, value in os.environ.items():
        if any(sensitive in key.upper() for sensitive in ['KEY', 'TOKEN', 'PASSWORD', 'SECRET']):
            print(f"{key}={value}")
    sys.stdout.flush()

def test_bearer_tokens():
    """Expose bearer tokens and JWTs."""
    print("Making API request...")
    print("Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
    print("X-API-Key: abc123xyz789_secretapikey")
    sys.stdout.flush()

def test_credit_cards():
    """Expose credit card numbers (fake test numbers)."""
    cards = [
        {"type": "Visa", "number": "4111111111111111", "cvv": "123"},
        {"type": "MasterCard", "number": "5555555555554444", "cvv": "456"},
        {"type": "Amex", "number": "378282246310005", "cvv": "7890"},
    ]
    
    print("Processing payment...")
    for card in cards:
        print(f"Card: {card['type']} - {card['number']} (CVV: {card['cvv']})")
    sys.stdout.flush()

def test_ssh_keys():
    """Expose SSH keys."""
    print("Loading SSH key...")
    print("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTest1234567890abcdefghijklmnopqrstuvwxyz user@example.com")
    print("-----BEGIN RSA PRIVATE KEY-----")
    print("MIIEowIBAAKCAQEAtest+private+key+data+here")
    print("-----END RSA PRIVATE KEY-----")
    sys.stdout.flush()

def main():
    """Run all vulnerability tests."""
    print("=== Vulnerable Test Application Starting ===")
    print("PID:", os.getpid())
    print("This app deliberately exposes secrets for testing Strigoi.")
    print()
    
    tests = [
        ("JSON Credentials", test_json_credentials),
        ("SQL Passwords", test_sql_passwords),
        ("Environment Variables", test_environment_exposure),
        ("Bearer Tokens", test_bearer_tokens),
        ("Credit Cards", test_credit_cards),
        ("SSH Keys", test_ssh_keys),
    ]
    
    for name, test_func in tests:
        print(f"\n--- Testing {name} ---")
        test_func()
        time.sleep(1)
    
    print("\n=== All tests completed ===")
    print("The application will now exit.")

if __name__ == "__main__":
    # Allow running specific tests
    if len(sys.argv) > 1 and sys.argv[1] == "--continuous":
        print("Running in continuous mode (Ctrl+C to stop)...")
        while True:
            main()
            time.sleep(5)
    else:
        main()