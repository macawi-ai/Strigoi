#!/bin/bash
# Creates a realistic SQLite database for demonstrating SQL injection privilege amplification

echo "=== Setting Up Enterprise Demo Database ==="
echo "[*] Creating SQLite database with sensitive enterprise data..."

# Remove existing database
rm -f enterprise-demo.db

# Create database with realistic sensitive data
sqlite3 enterprise-demo.db <<'EOF'
-- Customer table with PII
CREATE TABLE customers (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    ssn TEXT NOT NULL,
    credit_card TEXT NOT NULL,
    cvv TEXT NOT NULL,
    address TEXT NOT NULL,
    phone TEXT NOT NULL,
    account_balance DECIMAL(10,2),
    created_date DATE
);

INSERT INTO customers VALUES
(1, 'John Smith', 'john.smith@email.com', '123-45-6789', '4532-1234-5678-9012', '123', '123 Main St, Anytown, USA', '555-0101', 25000.50, '2023-01-15'),
(2, 'Sarah Johnson', 'sarah.j@email.com', '987-65-4321', '4111-1111-1111-1111', '456', '456 Oak Ave, City, USA', '555-0102', 75000.25, '2023-02-20'),
(3, 'Michael Davis', 'mdavis@email.com', '555-12-3456', '5555-5555-5555-4444', '789', '789 Pine St, Town, USA', '555-0103', 150000.00, '2023-03-10'),
(4, 'Emily Wilson', 'ewilson@email.com', '111-22-3333', '3782-822463-10005', '321', '321 Elm Dr, Village, USA', '555-0104', 45000.75, '2023-04-05'),
(5, 'Robert Brown', 'rbrown@email.com', '777-88-9999', '6011-1111-1111-1117', '654', '654 Maple Ln, Suburb, USA', '555-0105', 95000.30, '2023-05-12');

-- Employee table with sensitive HR data
CREATE TABLE employees (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    ssn TEXT NOT NULL,
    salary INTEGER NOT NULL,
    department TEXT NOT NULL,
    hire_date DATE,
    manager_id INTEGER,
    security_clearance TEXT
);

INSERT INTO employees VALUES
(1, 'Alice CEO', 'alice@company.com', '100-00-0001', 750000, 'Executive', '2020-01-01', NULL, 'TOP_SECRET'),
(2, 'Bob CTO', 'bob@company.com', '200-00-0002', 650000, 'Technology', '2020-06-15', 1, 'SECRET'),
(3, 'Carol CFO', 'carol@company.com', '300-00-0003', 600000, 'Finance', '2021-01-01', 1, 'SECRET'),
(4, 'David Engineer', 'david@company.com', '400-00-0004', 150000, 'Technology', '2021-03-15', 2, 'CONFIDENTIAL'),
(5, 'Eva Analyst', 'eva@company.com', '500-00-0005', 95000, 'Finance', '2021-06-01', 3, 'CONFIDENTIAL'),
(6, 'Frank Admin', 'frank@company.com', '600-00-0006', 65000, 'Operations', '2022-01-01', 1, 'PUBLIC'),
(7, 'Grace Security', 'grace@company.com', '700-00-0007', 120000, 'Security', '2022-03-01', 1, 'TOP_SECRET');

-- Financial transactions table
CREATE TABLE financial_transactions (
    id INTEGER PRIMARY KEY,
    customer_id INTEGER,
    transaction_type TEXT NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    description TEXT,
    transaction_date DATETIME,
    account_number TEXT,
    routing_number TEXT,
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

INSERT INTO financial_transactions VALUES
(1, 1, 'DEPOSIT', 5000.00, 'Salary deposit', '2024-01-15 09:30:00', '123456789', '021000021'),
(2, 1, 'WITHDRAWAL', -500.00, 'ATM withdrawal', '2024-01-16 14:22:00', '123456789', '021000021'),
(3, 2, 'WIRE_TRANSFER', 25000.00, 'Investment transfer', '2024-01-17 11:15:00', '987654321', '021000021'),
(4, 3, 'LOAN_PAYMENT', -2500.00, 'Mortgage payment', '2024-01-18 08:00:00', '555123456', '021000021'),
(5, 4, 'DEPOSIT', 3000.00, 'Bonus payment', '2024-01-19 16:45:00', '111222333', '021000021');

-- Executive compensation (SOX sensitive)
CREATE TABLE executive_compensation (
    id INTEGER PRIMARY KEY,
    executive_name TEXT NOT NULL,
    base_salary INTEGER NOT NULL,
    bonus INTEGER,
    stock_options INTEGER,
    other_compensation INTEGER,
    total_compensation INTEGER,
    year INTEGER
);

INSERT INTO executive_compensation VALUES
(1, 'Alice CEO', 750000, 1500000, 2000000, 250000, 4500000, 2024),
(2, 'Bob CTO', 650000, 800000, 1200000, 150000, 2800000, 2024),
(3, 'Carol CFO', 600000, 750000, 1000000, 100000, 2450000, 2024);

-- Audit logs (that attackers will want to hide their tracks)
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY,
    user_name TEXT NOT NULL,
    action TEXT NOT NULL,
    table_name TEXT,
    record_id INTEGER,
    timestamp DATETIME,
    ip_address TEXT,
    success BOOLEAN
);

INSERT INTO audit_logs VALUES
(1, 'admin', 'LOGIN', NULL, NULL, '2024-01-20 08:00:00', '192.168.1.100', 1),
(2, 'admin', 'SELECT', 'customers', NULL, '2024-01-20 08:05:00', '192.168.1.100', 1),
(3, 'finance_user', 'SELECT', 'financial_transactions', NULL, '2024-01-20 09:00:00', '192.168.1.105', 1),
(4, 'hr_user', 'SELECT', 'employees', NULL, '2024-01-20 10:00:00', '192.168.1.110', 1);

-- Trade secrets table
CREATE TABLE trade_secrets (
    id INTEGER PRIMARY KEY,
    product_name TEXT NOT NULL,
    formula TEXT NOT NULL,
    manufacturing_cost DECIMAL(8,2),
    market_value DECIMAL(10,2),
    classification TEXT
);

INSERT INTO trade_secrets VALUES
(1, 'SuperWidget Pro', 'C8H10N4O2 + proprietary catalyst X-47', 12.50, 299.99, 'TOP_SECRET'),
(2, 'MegaGadget Elite', 'Titanium alloy blend: Ti-6Al-4V + secret element Y', 45.00, 1299.99, 'SECRET'),
(3, 'UltraDevice Max', 'Quantum processing algorithm v3.7.2', 0.25, 99.99, 'CONFIDENTIAL');

EOF

echo "[+] Database created: enterprise-demo.db"
echo "[+] Tables created:"
echo "    - customers (5 records with PII, credit cards)"
echo "    - employees (7 records with salaries, SSNs, clearances)"
echo "    - financial_transactions (5 records with account numbers)"
echo "    - executive_compensation (3 records, SOX-sensitive)"
echo "    - audit_logs (4 records, attackers will want to clear)"
echo "    - trade_secrets (3 records with proprietary formulas)"
echo
echo "[*] Database contains realistic sensitive data for demonstration"
echo "[*] File size: $(du -h enterprise-demo.db | cut -f1)"
echo
echo "[!] This is MOCK data for security research only!"
echo "[!] Do not use real credentials or PII in demonstrations!"