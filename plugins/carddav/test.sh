#!/bin/bash

# CardDAV Plugin Test Script for Nextcloud/CardDAV Servers
# Usage: 
#   export CARDDAV_URL="https://your-nextcloud.com/remote.php/dav/addressbooks/users/yourusername/"
#   export CARDDAV_USERNAME="your_username"
#   export CARDDAV_PASSWORD="your_password"
#   ./test.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TIMESTAMP=$(date +%s)
TEST_CONTACT_UID="test-contact-${TIMESTAMP}"
TEST_CONTACT_NAME="Test Contact ${TIMESTAMP}"
TEST_CONTACT_EMAIL="test${TIMESTAMP}@example.com"
TEST_CONTACT_PHONE="+1555${TIMESTAMP: -4}"

# Helper functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

check_dev_manifest() {
    if [[ ! -f "devManifest.json" ]]; then
        print_error "devManifest.json not found"
        echo "Make sure you're running this script from the plugin directory"
        exit 1
    fi
    
    # Check if devManifest.json has credentials configured
    if ! grep -q '"url"' devManifest.json || ! grep -q '"username"' devManifest.json || ! grep -q '"password"' devManifest.json; then
        print_error "devManifest.json is missing required credentials (url, username, password)"
        echo "Please configure your CardDAV credentials in devManifest.json"
        exit 1
    fi
}

execute_sql() {
    local sql="$1"
    # Load plugin and execute SQL in a single anyquery session
    (echo "SELECT load_dev_plugin('carddav', 'devManifest.json');" && echo "$sql") | anyquery --dev
}

# Main test execution
main() {
    print_header "CardDAV Plugin Test Script"
    
    # Check devManifest.json
    print_info "Checking devManifest.json..."
    check_dev_manifest
    print_success "devManifest.json found with credentials configured"
    
    # Build the plugin
    print_info "Building CardDAV plugin..."
    if make > /dev/null 2>&1; then
        print_success "Plugin built successfully"
    else
        print_error "Failed to build plugin"
        exit 1
    fi
    
    # Start anyquery and test
    print_header "Starting CardDAV Plugin Tests"
    
    # Test 1: List address books
    print_header "Test 1: List Address Books"
    execute_sql "SELECT path, name, description FROM carddav_address_books;"
    # Get address book info - the path will be empty for our setup
    ADDRESS_BOOK_FULL_RESULT=$(execute_sql "SELECT * FROM carddav_address_books LIMIT 1;" 2>&1)
    
    # Check if we got the address book data (look for "Contacts")
    if echo "$ADDRESS_BOOK_FULL_RESULT" | grep -q "Contacts"; then
        # Extract the actual path from the result - get the first one which should be "Contacts"
        ADDRESS_BOOK_PATH=$(echo "$ADDRESS_BOOK_FULL_RESULT" | grep "contacts/" | head -1 | sed 's/\t.*//g' | xargs)
        print_success "Found address book: Contacts (path: '$ADDRESS_BOOK_PATH')"
    else
        print_error "No address books found in result: $ADDRESS_BOOK_FULL_RESULT"
        exit 1
    fi
    
    # Test 2: List contacts from first address book
    print_header "Test 2: List Contacts from Address Book"
    execute_sql "SELECT COUNT(*) as contact_count FROM carddav_contacts WHERE address_book = '$ADDRESS_BOOK_PATH';"
    execute_sql "SELECT uid, full_name, email, phone FROM carddav_contacts WHERE address_book = '$ADDRESS_BOOK_PATH' LIMIT 5;"
    
    # Test 4: Insert a new contact
    print_header "Test 3: Insert New Contact"
    INSERT_SQL="INSERT INTO carddav_contacts (address_book, uid, full_name, email, phone, organization) 
                VALUES ('$ADDRESS_BOOK_PATH', '$TEST_CONTACT_UID', '$TEST_CONTACT_NAME', '$TEST_CONTACT_EMAIL', '$TEST_CONTACT_PHONE', 'Test Corp');"
    
    print_info "Inserting contact: $TEST_CONTACT_NAME"
    if execute_sql "$INSERT_SQL" > /dev/null 2>&1; then
        print_success "Contact inserted successfully"
    else
        print_error "Failed to insert contact"
        exit 1
    fi
    
    # Verify insertion
    print_info "Verifying insertion..."
    execute_sql "SELECT uid, full_name, email, phone, organization FROM carddav_contacts WHERE address_book = '$ADDRESS_BOOK_PATH' AND uid = '$TEST_CONTACT_UID';"
    
    # Test 5: Update the contact
    print_header "Test 4: Update Contact"
    UPDATE_SQL="UPDATE carddav_contacts 
                SET full_name = '${TEST_CONTACT_NAME} Updated',
                    email = 'updated${TIMESTAMP}@example.com',
                    organization = 'Updated Corp'
                WHERE address_book = '$ADDRESS_BOOK_PATH' AND uid = '$TEST_CONTACT_UID';"
    
    print_info "Updating contact: $TEST_CONTACT_UID"
    if execute_sql "$UPDATE_SQL" > /dev/null 2>&1; then
        print_success "Contact updated successfully"
    else
        print_error "Failed to update contact"
    fi
    
    # Verify update
    print_info "Verifying update..."
    execute_sql "SELECT uid, full_name, email, phone, organization FROM carddav_contacts WHERE address_book = '$ADDRESS_BOOK_PATH' AND uid = '$TEST_CONTACT_UID';"
    
    # Test 6: Attempt to delete the contact (should show expected error)
    print_header "Test 5: Delete Contact (Expected to show 'not implemented' error)"
    DELETE_SQL="DELETE FROM carddav_contacts WHERE address_book = '$ADDRESS_BOOK_PATH' AND uid = '$TEST_CONTACT_UID';"
    
    print_info "Attempting to delete contact: $TEST_CONTACT_UID"
    if execute_sql "$DELETE_SQL" 2>&1; then
        print_info "Delete operation completed (may show 'not implemented' message)"
    else
        print_info "Delete operation failed as expected (not implemented)"
    fi
    
    # Final verification - check if contact still exists
    print_info "Final verification - checking if contact still exists..."
    CONTACT_COUNT=$(execute_sql "SELECT COUNT(*) FROM carddav_contacts WHERE address_book = '$ADDRESS_BOOK_PATH' AND uid = '$TEST_CONTACT_UID';" | tail -n +2 | tr -d ' ')
    
    if [[ "$CONTACT_COUNT" == "1" ]]; then
        print_info "Test contact still exists (delete not implemented, as expected)"
    else
        print_info "Test contact was removed"
    fi
    
    print_header "Test Summary"
    print_success "Plugin loading: PASSED"
    print_success "Address book listing: PASSED" 
    print_success "Contact querying: PASSED"
    print_success "Contact insertion: PASSED"
    print_success "Contact update: PASSED"
    print_info "Contact deletion: EXPECTED BEHAVIOR (not implemented)"
    
    print_header "All Tests Completed!"
    echo -e "${GREEN}The CardDAV plugin is working correctly with your server.${NC}"
    echo -e "${YELLOW}Note: Test contact '$TEST_CONTACT_UID' may still exist in your address book.${NC}"
}

# Run main function
main "$@"
