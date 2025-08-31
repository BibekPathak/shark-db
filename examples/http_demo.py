#!/usr/bin/env python3
"""
sharkDB HTTP API Demo
This script demonstrates how to use sharkDB's HTTP REST API
"""

import requests
import json
import time

BASE_URL = "http://localhost:8090"

def print_response(response, description):
    print(f"\n{description}:")
    print(f"Status: {response.status_code}")
    if response.text:
        print(f"Response: {response.text.strip()}")
    print("-" * 50)

def demo_http_api():
    print("🐋 sharkDB HTTP API Demo")
    print("=" * 50)
    
    # Start with clean state
    print("Starting HTTP API demo...")
    
    # Demo 1: Table Management
    print("\n1. Table Management")
    print("-" * 30)
    
    # List tables (should be empty initially)
    response = requests.get(f"{BASE_URL}/tables")
    print_response(response, "List tables (initial)")
    
    # Create a table
    response = requests.post(f"{BASE_URL}/tables", params={"name": "users"})
    print_response(response, "Create users table")
    
    # Create another table
    response = requests.post(f"{BASE_URL}/tables", params={"name": "products"})
    print_response(response, "Create products table")
    
    # List tables again
    response = requests.get(f"{BASE_URL}/tables")
    print_response(response, "List tables (after creation)")
    
    # Demo 2: Key-Value Operations
    print("\n2. Key-Value Operations")
    print("-" * 30)
    
    # Insert user data
    user_data = {"name": "Alice", "age": 25, "email": "alice@example.com"}
    response = requests.put(f"{BASE_URL}/kv/users/alice", json=user_data)
    print_response(response, "Insert user 'alice'")
    
    # Insert more users
    users = [
        ("bob", {"name": "Bob", "age": 30, "email": "bob@example.com"}),
        ("charlie", {"name": "Charlie", "age": 35, "email": "charlie@example.com"}),
        ("dave", {"name": "Dave", "age": 28, "email": "dave@example.com"})
    ]
    
    for username, data in users:
        response = requests.put(f"{BASE_URL}/kv/users/{username}", json=data)
        print_response(response, f"Insert user '{username}'")
    
    # Get a user
    response = requests.get(f"{BASE_URL}/kv/users/alice")
    print_response(response, "Get user 'alice'")
    
    # Update a user
    updated_data = {"name": "Alice Smith", "age": 26, "email": "alice.smith@example.com"}
    response = requests.put(f"{BASE_URL}/kv/users/alice", json=updated_data)
    print_response(response, "Update user 'alice'")
    
    # Verify update
    response = requests.get(f"{BASE_URL}/kv/users/alice")
    print_response(response, "Get updated user 'alice'")
    
    # Demo 3: Scanning
    print("\n3. Scanning Operations")
    print("-" * 30)
    
    # Scan all users
    response = requests.get(f"{BASE_URL}/scan/users")
    print_response(response, "Scan all users")
    
    # Scan with limit
    response = requests.get(f"{BASE_URL}/scan/users", params={"limit": 2})
    print_response(response, "Scan users with limit=2")
    
    # Scan from specific key
    response = requests.get(f"{BASE_URL}/scan/users", params={"start": "bob"})
    print_response(response, "Scan users from 'bob'")
    
    # Prefix scan
    response = requests.get(f"{BASE_URL}/prefix/users", params={"prefix": "a"})
    print_response(response, "Prefix scan users starting with 'a'")
    
    # Demo 4: Statistics
    print("\n4. Statistics")
    print("-" * 30)
    
    # Get table statistics
    response = requests.get(f"{BASE_URL}/stats/users")
    print_response(response, "Get users table statistics")
    
    # Demo 5: Product Data
    print("\n5. Product Data")
    print("-" * 30)
    
    # Insert product data
    products = [
        ("laptop", {"name": "MacBook Pro", "price": 1299, "category": "electronics"}),
        ("phone", {"name": "iPhone 15", "price": 799, "category": "electronics"}),
        ("book", {"name": "Database Design", "price": 49, "category": "books"}),
        ("tablet", {"name": "iPad Air", "price": 599, "category": "electronics"})
    ]
    
    for product_id, data in products:
        response = requests.put(f"{BASE_URL}/kv/products/{product_id}", json=data)
        print_response(response, f"Insert product '{product_id}'")
    
    # Scan products
    response = requests.get(f"{BASE_URL}/scan/products")
    print_response(response, "Scan all products")
    
    # Demo 6: Error Handling
    print("\n6. Error Handling")
    print("-" * 30)
    
    # Try to get non-existent key
    response = requests.get(f"{BASE_URL}/kv/users/nonexistent")
    print_response(response, "Get non-existent user")
    
    # Try to get from non-existent table
    response = requests.get(f"{BASE_URL}/kv/nonexistent/key")
    print_response(response, "Get from non-existent table")
    
    # Demo 7: Cleanup
    print("\n7. Cleanup")
    print("-" * 30)
    
    # Delete a user
    response = requests.delete(f"{BASE_URL}/kv/users/dave")
    print_response(response, "Delete user 'dave'")
    
    # Verify deletion
    response = requests.get(f"{BASE_URL}/kv/users/dave")
    print_response(response, "Get deleted user 'dave'")
    
    # Drop a table
    response = requests.delete(f"{BASE_URL}/tables/products")
    print_response(response, "Drop products table")
    
    # Try to access dropped table
    response = requests.get(f"{BASE_URL}/kv/products/laptop")
    print_response(response, "Access dropped table")
    
    print("\n🐋 HTTP API Demo Complete!")
    print("=" * 50)

if __name__ == "__main__":
    try:
        demo_http_api()
    except requests.exceptions.ConnectionError:
        print("❌ Error: Could not connect to sharkDB HTTP server.")
        print("Make sure sharkDB is running with: ./sharkdb -http :8090")
    except Exception as e:
        print(f"❌ Error: {e}")
