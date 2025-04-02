#!/bin/bash
#
# AdGuard Private DDNS Update Script
# For Linux/macOS systems
#

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration - modify before running
base_url="{{server_name}}" # Example: http://localhost:34020 or https://dns.example.com
username="{{username}}"    # AdGuard Private username
password="{{password}}"    # AdGuard Private password
domain="{{domain}}"        # Domain to update, e.g.: nas.example.com
cookies="{{cookies}}"      # Cookie string for authentication, e.g.: "agh_session=abc123"

# DDNS Configuration
enable_ipv4="true" # Enable IPv4 DDNS updates
enable_ipv6="true" # Enable IPv6 DDNS updates

# WARNING: Cookies may expire over time, which could cause authentication failures.
# It is recommended to use username/password authentication whenever possible.
# Only use cookies if username/password authentication is not available.

# Display usage information
show_usage() {
    echo -e "${BLUE}Usage:${NC}"
    echo -e "  Edit the script and set the following parameters before running:"
    echo -e "  ${YELLOW}base_url${NC}  - AdGuard Private server URL (e.g., https://{xxxxxxxxxxxxxxxx}.adguardprivate.com)"
    echo -e "  ${YELLOW}domain${NC}    - Domain to update (e.g., nas.example.com)"
    echo -e ""
    echo -e "  For authentication, use one of the following methods:"
    echo -e "  1. Username/Password (recommended):"
    echo -e "     ${YELLOW}username${NC} - AdGuard Private username"
    echo -e "     ${YELLOW}password${NC} - AdGuard Private password"
    echo -e ""
    echo -e "  2. Cookies (alternative, may expire):"
    echo -e "     ${YELLOW}cookies${NC}  - Cookie string (e.g., \"agh_session=abc123\")"
    echo -e ""
    echo -e "  Example configuration:"
    echo -e "    base_url=\"https://{xxxxxxxxxxxxxxxx}.adguardprivate.com\""
    echo -e "    username=\"admin\""
    echo -e "    password=\"password123\""
    echo -e "    domain=\"nas.example.com\""
    echo -e ""
    echo -e "    # OR using cookies instead of username/password:"
    echo -e "    cookies=\"agh_session=abc123\""
    echo -e ""
    echo -e "${BLUE}Note:${NC} This script is specifically developed for adguardprivate.com"
    exit 1
}

# Print curl command function
print_curl_cmd() {
    echo -e "${YELLOW}[CURL] $1${NC}"
}

# Check dependencies
check_dependencies() {
    DEPS=("curl" "grep" "awk")
    MISSING=0

    for dep in "${DEPS[@]}"; do
        if ! command -v "$dep" &>/dev/null; then
            echo -e "${RED}Error: Required program '$dep' not found${NC}"
            MISSING=1
        fi
    done

    if [ $MISSING -eq 1 ]; then
        echo -e "${RED}Please install missing dependencies and retry${NC}"
        exit 1
    fi
}

# Get current public IP addresses
get_current_ip() {
    local ip_version="$1"
    local curl_opts=""
    local services=()

    if [ "$ip_version" = "ipv4" ]; then
        curl_opts="-4"
        services=(
            "https://api.ipify.org"
            "https://ifconfig.me/ip"
            "https://icanhazip.com"
            "https://ipinfo.io/ip"
        )
    else
        curl_opts="-6"
        services=(
            "https://api6.ipify.org"
            "https://ifconfig.co"
            "https://icanhazip.com"
            "https://api6.my-ip.io/ip"
        )
    fi

    for service in "${services[@]}"; do
        current_ip=$(curl -s $curl_opts "$service")

        if [ "$ip_version" = "ipv4" ]; then
            if [[ $current_ip =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                echo "$current_ip"
                return 0
            fi
        else
            if [[ $current_ip =~ ^([0-9a-fA-F]{0,4}:){1,7}[0-9a-fA-F]{0,4}$ ]]; then
                echo "$current_ip"
                return 0
            fi
        fi
    done

    echo ""
    return 1
}

# Generate auth header
get_auth_header() {
    if [ -n "$username" ] && [ -n "$password" ]; then
        auth_base64=$(echo -n "$username:$password" | base64)
        echo "Authorization: Basic $auth_base64"
    elif [ -n "$cookies" ]; then
        echo "Cookie: $cookies"
    else
        echo ""
    fi
}

# Get curl options for API access
get_curl_opts() {
    if [ $DEBUG -eq 1 ]; then
        echo "-S --connect-timeout 10 --max-time 15"
    else
        echo "-sS --connect-timeout 10 --max-time 15"
    fi
}

# Get all DNS records
get_all_dns_records() {
    local auth_header=$(get_auth_header)
    local CURL_OPTS=$(get_curl_opts)

    if [ -z "$auth_header" ]; then
        echo -e "${RED}Error: No authentication method available${NC}"
        exit 1
    fi

    print_curl_cmd "curl $CURL_OPTS -H \"$auth_header\" ${base_url}/control/rewrite/list"
    response=$(curl $CURL_OPTS -H "$auth_header" ${base_url}/control/rewrite/list)

    if [[ $response == *"401"* || $response == *"Unauthorized"* ]]; then
        echo -e "${RED}Error: Authentication failed. If using cookies, they may have expired.${NC}"
        echo -e "${RED}Please update your authentication information and try again.${NC}"
        exit 1
    fi

    echo "$response"
}

# Verify if a specific record exists with expected values
verify_dns_record() {
    local ip="$1"
    local ip_version="$2"
    local all_records="$3"

    echo -e "${YELLOW}Examining all records for domain '${domain}' with IP '$ip'${NC}"

    # Print debugging information about the record format
    echo -e "${YELLOW}Record response content:${NC}"
    echo "$all_records" | head -30

    # Check if response is valid JSON or API response
    if [[ "$all_records" == *"\"result\""* ]]; then
        echo -e "${RED}API Response format detected instead of records. Response:${NC}"
        echo "$all_records"
    fi

    # Extract records for the specific domain
    domain_records=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}")
    echo -e "${YELLOW}Domain records found: $(echo "$domain_records" | wc -l)${NC}"
    echo "$domain_records"

    if [ "$ip_version" = "ipv4" ]; then
        # For IPv4, look for records with exact IPv4 format
        record_exists=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
            grep -o "{[^}]*\"answer\":\"$ip\"[^}]*}" | head -1)
    else
        # For IPv6, look for records with the IPv6 address
        record_exists=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
            grep -o "{[^}]*\"answer\":\"$ip\"[^}]*}" | head -1)
    fi

    if [ -n "$record_exists" ]; then
        echo "true"
    else
        echo "false"
    fi
}

# Get current DNS record for domain
get_current_record() {
    local ip_version="$1"
    local auth_header=$(get_auth_header)
    local CURL_OPTS=$(get_curl_opts)

    if [ -z "$auth_header" ]; then
        echo -e "${RED}Error: No authentication method available${NC}"
        exit 1
    fi

    print_curl_cmd "curl $CURL_OPTS -H \"$auth_header\" ${base_url}/control/rewrite/list"
    response=$(curl $CURL_OPTS -H "$auth_header" ${base_url}/control/rewrite/list)

    if [[ $response == *"401"* || $response == *"Unauthorized"* ]]; then
        echo -e "${RED}Error: Authentication failed. If using cookies, they may have expired.${NC}"
        echo -e "${RED}Please update your authentication information and try again.${NC}"
        exit 1
    fi

    # Check if this is a valid JSON response
    if [[ $response == "400 Bad Request" ]]; then
        echo ""
        return 1
    fi

    # If this is an AdGuard Private response with the items field (newer versions)
    if [[ $response == *"\"items\""* ]]; then
        if [[ $response == *"\"domain\":\"${domain}\""* ]]; then
            # Filter records based on IP version
            if [ "$ip_version" = "ipv4" ]; then
                # Find record with IPv4 answer (matches x.x.x.x format)
                current_record=$(echo "$response" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" | grep -o "{[^}]*\"answer\":\"[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+\"[^}]*}" | head -1)
            else
                # Find record with IPv6 answer (matches hex format with colons)
                current_record=$(echo "$response" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" | grep -o "{[^}]*\"answer\":\"[0-9a-fA-F:]\+\"[^}]*}" | grep -v "[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+" | head -1)
            fi
            echo "$current_record"
            return 0
        else
            echo ""
            return 1
        fi
    else
        # For direct record listing in older versions
        if [[ $response == *"\"domain\":\"${domain}\""* ]]; then
            # Filter records based on IP version
            if [ "$ip_version" = "ipv4" ]; then
                # Find record with IPv4 answer (matches x.x.x.x format)
                current_record=$(echo "$response" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" | grep -o "{[^}]*\"answer\":\"[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+\"[^}]*}" | head -1)
            else
                # Find record with IPv6 answer (matches hex format with colons)
                current_record=$(echo "$response" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" | grep -o "{[^}]*\"answer\":\"[0-9a-fA-F:]\+\"[^}]*}" | grep -v "[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+" | head -1)
            fi
            echo "$current_record"
            return 0
        else
            echo ""
            return 1
        fi
    fi
}

# Delete existing DNS record
delete_existing_record() {
    local record="$1"
    local auth_header=$(get_auth_header)
    local CURL_OPTS=$(get_curl_opts)
    local deleted_ip=$(echo "$record" | grep -o '"answer":"[^"]*"' | cut -d'"' -f4)

    print_curl_cmd "curl $CURL_OPTS -X POST -H \"Content-Type: application/json\" -H \"$auth_header\" -d \"$record\" ${base_url}/control/rewrite/delete"
    curl $CURL_OPTS -X POST -H "Content-Type: application/json" \
        -H "$auth_header" \
        -d "$record" \
        ${base_url}/control/rewrite/delete >/dev/null 2>&1

    # Wait a moment for the changes to propagate
    sleep 1

    # Check if the record was actually deleted by querying the list API
    local all_records=$(get_all_dns_records)

    # Look for the record in the current list
    local record_exists=""
    if [[ "$record" == *"\"answer\":\"[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+\""* ]]; then
        # IPv4 record
        record_exists=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
            grep -o "{[^}]*\"answer\":\"$deleted_ip\"[^}]*}" | head -1)
    else
        # IPv6 record
        record_exists=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
            grep -o "{[^}]*\"answer\":\"$deleted_ip\"[^}]*}" | head -1)
    fi

    if [ -z "$record_exists" ]; then
        echo -e "${GREEN}Verified: DNS record was deleted successfully${NC}"
        return 0
    else
        echo -e "${RED}Failed to delete DNS record: Record still exists in the list${NC}"
        return 1
    fi
}

# Create new DNS record
create_new_record() {
    local new_ip="$1"
    local ip_version="$2"
    local auth_header=$(get_auth_header)
    local CURL_OPTS=$(get_curl_opts)

    print_curl_cmd "curl $CURL_OPTS -X POST -H \"Content-Type: application/json\" -H \"$auth_header\" -d \"{\\\"domain\\\": \\\"${domain}\\\", \\\"answer\\\": \\\"$new_ip\\\"}\" ${base_url}/control/rewrite/add"
    # Execute the curl command but ignore the response as the API doesn't return meaningful data
    curl $CURL_OPTS -X POST \
        -H "Content-Type: application/json" \
        -H "$auth_header" \
        -d "{\"domain\": \"${domain}\", \"answer\": \"$new_ip\"}" \
        ${base_url}/control/rewrite/add >/dev/null 2>&1

    echo -e "${GREEN}Created new DNS record: ${domain} -> $new_ip${NC}"

    # Wait a moment for the changes to propagate
    sleep 1

    # List all records to check if our record was set correctly
    local all_records=$(get_all_dns_records)

    # Print the response from get_all_dns_records
    echo -e "${CYAN}All DNS records:${NC}"
    echo "$all_records"

    echo -e "${CYAN}Checking if new record was created correctly...${NC}"

    # Parse the records to find our new record
    if [[ "$all_records" == *"\"domain\":\"${domain}\""* ]]; then
        if [ "$ip_version" = "ipv4" ]; then
            # Look for IPv4 record matching our domain and IP
            verify_record=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
                grep -o "{[^}]*\"answer\":\"$new_ip\"[^}]*}" | head -1)
        else
            # Look for IPv6 record matching our domain and IP
            verify_record=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
                grep -o "{[^}]*\"answer\":\"$new_ip\"[^}]*}" | head -1)
        fi

        if [ ! -z "$verify_record" ]; then
            echo -e "${GREEN}Verified: Record was created successfully: ${domain} -> $new_ip${NC}"
            return 0
        else
            echo -e "${RED}Failed to verify record creation: Record not found in the list.${NC}"
            return 1
        fi
    else
        echo -e "${RED}Failed to verify record creation: Domain not found in records.${NC}"
        return 1
    fi
}

# Update DNS record for specific IP version
update_dns_record() {
    local ip_version="$1"
    local current_ip=""

    # Get current IP address
    current_ip=$(get_current_ip "$ip_version")
    if [ -z "$current_ip" ]; then
        echo -e "${YELLOW}Warning: Unable to get current $ip_version address, skipping $ip_version update${NC}"
        return 1
    fi
    echo -e "Current $ip_version: ${GREEN}$current_ip${NC}"

    # Get all DNS records
    all_records=$(get_all_dns_records)

    # Print the response from get_all_dns_records
    echo -e "${CYAN}All DNS records:${NC}"
    echo "$all_records"

    # Check if API returned an error
    if [[ "$all_records" == "400 Bad Request" || "$all_records" == *"\"error\""* ]]; then
        echo -e "${RED}Error: API returned error when fetching records${NC}"
        # Try to create a new record without verification
        create_new_record "$current_ip" "$ip_version"
        return $?
    fi

    # Check if domain exists in records
    local domain_exists=false
    local existing_record=""
    local existing_ip=""

    if [[ "$all_records" == *"\"domain\":\"${domain}\""* ]]; then
        domain_exists=true

        # Search for the specific IP version record
        if [ "$ip_version" = "ipv4" ]; then
            # Find record with IPv4 answer (matches x.x.x.x format)
            existing_record=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
                grep -o "{[^}]*\"answer\":\"[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+\"[^}]*}" | head -1)
        else
            # Find record with IPv6 answer (matches hex format with colons)
            existing_record=$(echo "$all_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
                grep -o "{[^}]*\"answer\":\"[0-9a-fA-F:]\+\"[^}]*}" |
                grep -v "[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+" | head -1)
        fi

        if [ ! -z "$existing_record" ]; then
            # Extract the current IP from existing record
            existing_ip=$(echo "$existing_record" | grep -o '"answer":"[^"]*"' | cut -d'"' -f4)

            echo -e "Found existing ${ip_version} record: ${domain} -> ${existing_ip}"

            # Compare IPs
            if [ "$current_ip" = "$existing_ip" ]; then
                echo -e "${GREEN}DNS record is up to date (${domain} -> $current_ip)${NC}"
                return 0
            fi

            # Update existing record with new IP
            echo -e "Updating ${YELLOW}$domain${NC} record from $existing_ip to $current_ip..."
            local auth_header=$(get_auth_header)
            local CURL_OPTS=$(get_curl_opts)

            # Format the JSON correctly
            local update_json="{\"target\": $existing_record, \"update\": {\"domain\": \"${domain}\", \"answer\": \"$current_ip\"}}"

            print_curl_cmd "curl $CURL_OPTS -X PUT -H \"Content-Type: application/json\" -H \"$auth_header\" -d '$update_json' ${base_url}/control/rewrite/update"

            # Execute the curl command but ignore the response
            curl $CURL_OPTS -X PUT \
                -H "Content-Type: application/json" \
                -H "$auth_header" \
                -d "$update_json" \
                ${base_url}/control/rewrite/update >/dev/null 2>&1

            # Wait a moment for the changes to propagate
            sleep 1

            # List all records to check if our record was updated correctly
            local updated_records=$(get_all_dns_records)

            echo -e "${CYAN}Checking if record was updated correctly...${NC}"

            # Look for our specific record in the updated records
            if [ "$ip_version" = "ipv4" ]; then
                # Look for IPv4 record matching our domain and new IP
                verify_record=$(echo "$updated_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
                    grep -o "{[^}]*\"answer\":\"$current_ip\"[^}]*}" | head -1)
            else
                # Look for IPv6 record matching our domain and new IP
                verify_record=$(echo "$updated_records" | grep -o "{[^}]*\"domain\":\"${domain}\"[^}]*}" |
                    grep -o "{[^}]*\"answer\":\"$current_ip\"[^}]*}" | head -1)
            fi

            if [ ! -z "$verify_record" ]; then
                echo -e "${GREEN}Verified: Record was updated successfully: ${domain} -> $current_ip${NC}"
                return 0
            else
                echo -e "${RED}Failed to verify record update: Record not found in the list.${NC}"
                return 1
            fi
        else
            echo -e "Domain $domain exists but ${YELLOW}has no ${ip_version} record yet${NC}"
            # Create a new record for this IP version
            create_new_record "$current_ip" "$ip_version"
            return $?
        fi
    else
        echo -e "Domain $domain ${YELLOW}does not exist in DNS records${NC}"
        # Create a new record for this domain
        create_new_record "$current_ip" "$ip_version"
        return $?
    fi
}

# Check if all essential parameters are provided
check_params() {
    local missing=0

    # Check base_url
    if [ -z "$base_url" ]; then
        echo -e "${RED}Error: Server URL (base_url) is required${NC}"
        missing=1
    fi

    # Check domain
    if [ -z "$domain" ]; then
        echo -e "${RED}Error: Domain name (domain) is required${NC}"
        missing=1
    fi

    # Check authentication
    if [ -z "$username" ] || [ -z "$password" ]; then
        if [ -z "$cookies" ]; then
            echo -e "${RED}Error: Authentication is required (either username/password or cookies)${NC}"
            missing=1
        fi
    fi

    # Show usage if any essential parameter is missing
    if [ $missing -eq 1 ]; then
        echo -e ""
        show_usage
    fi
}

# Check authentication method
check_auth() {
    if [ -z "$username" ] || [ -z "$password" ]; then
        if [ -z "$cookies" ]; then
            echo -e "${RED}Error: No authentication method available.${NC}"
            echo -e "${RED}Please provide either username/password or cookies.${NC}"
            show_usage
        else
            echo -e "${YELLOW}Warning: Using cookies for authentication. Username/password is recommended as cookies may expire.${NC}"
        fi
    fi
}

# Main function
main() {
    echo -e "${BLUE}AdGuard Private DDNS Update Script${NC}"
    echo -e "${BLUE}=============================${NC}"

    # Check if all essential parameters are provided
    check_params

    # Check authentication method
    check_auth

    # Check required dependencies
    check_dependencies

    # Update IPv4 record if enabled
    if [ "$enable_ipv4" = "true" ]; then
        echo -e "\n${BLUE}Updating IPv4 record${NC}"
        echo -e "${BLUE}------------------${NC}"
        update_dns_record "ipv4"
    else
        echo -e "\n${YELLOW}IPv4 updates disabled${NC}"
    fi

    # Update IPv6 record if enabled
    if [ "$enable_ipv6" = "true" ]; then
        echo -e "\n${BLUE}Updating IPv6 record${NC}"
        echo -e "${BLUE}------------------${NC}"
        update_dns_record "ipv6"
    else
        echo -e "\n${YELLOW}IPv6 updates disabled${NC}"
    fi

    echo -e "\n${GREEN}DDNS update completed${NC}"
}

# Execute main function
main "$@"
