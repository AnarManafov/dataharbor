#!/bin/bash
# ============================================
# Setup Test Data for User Mapping Demo
# ============================================
# Creates test files owned by different users
# to demonstrate token-to-user mapping
# ============================================

set -e

echo "Setting up test data for user mapping demonstration..."

# Read CREATE_MANY_FILES from environment (default to false)
CREATE_MANY_FILES=${CREATE_MANY_FILES:-false}

# Create test files for each user
create_test_files() {
    local user=$1
    local user_dir="/data/${user}"
    
    if [ -d "$user_dir" ]; then
        echo "Creating test files for user: ${user}"
        
        # Create as root, then change ownership
        cat > "$user_dir/README.txt" <<EOF
This directory belongs to user: ${user}

Files in this directory can only be accessed when:
- Your JWT token's 'sub' claim is mapped to '${user}' in the mapfile
- The SciTokens plugin successfully maps your identity

Test the mapping by:
1. Login with your token (sub claim in JWT)
2. Browse to /data/${user}
3. If you see this file, mapping worked!
EOF
        
        # Create a sample data file
        echo "Sample data for ${user}" > "$user_dir/sample.txt"
        echo "Timestamp: $(date)" >> "$user_dir/sample.txt"
        
        # Create additional test files for amanafov user
        if [ "$user" = "amanafov" ]; then
            # Create 2 random data files (10-15 MB each) for download testing
            # Using dd with /dev/urandom for efficient random data generation
            echo "Generating test data files..."
            dd if=/dev/urandom of="$user_dir/testfile_10MB.bin" bs=1M count=10 2>/dev/null
            dd if=/dev/urandom of="$user_dir/testfile_15MB.bin" bs=1M count=15 2>/dev/null
            
            # Create many_files subdirectory to test browsing with many files (if enabled)
            if [ "$CREATE_MANY_FILES" = "true" ]; then
                echo "Creating many_files subdirectory with 2K files for browsing test..."
                mkdir -p "$user_dir/many_files"
                
                # Generate one larger random file to split
                dd if=/dev/urandom of="$user_dir/many_files/random_data.tmp" bs=1K count=4000 2>/dev/null
                
                # Split into 2K files of ~2KB each
                cd "$user_dir/many_files"
                split -b 2048 random_data.tmp file_ --numeric-suffixes=1 --suffix-length=4
                rm random_data.tmp
                
                # Rename files to have .bin extension and add some variety
                local counter=1
                for file in file_*; do
                    case $((counter % 4)) in
                        0) mv "$file" "data_${counter}.bin" ;;
                        1) mv "$file" "test_${counter}.txt" ;;
                        2) mv "$file" "sample_${counter}.dat" ;;
                        3) mv "$file" "file_${counter}.log" ;;
                    esac
                    counter=$((counter + 1))
                done
                
                cd - > /dev/null
                echo "  [+] Created 2000 files in $user_dir/many_files"
            else
                echo "  [i] Skipping many_files creation (CREATE_MANY_FILES=false)"
            fi
        fi
        
        # Set ownership
        chown -R ${user}:${user} "$user_dir"
        chmod 700 "$user_dir"
        
        # Set proper permissions for many_files subdirectory and its contents first
        if [ -d "$user_dir/many_files" ]; then
            chmod 700 "$user_dir/many_files"
            chmod 644 "$user_dir/many_files"/*
        fi
        
        # Set permissions for files (not directories) in user_dir
        find "$user_dir" -maxdepth 1 -type f -exec chmod 644 {} \;
        
        echo "  [+] Created files in $user_dir"
    else
        echo "  [!] Directory $user_dir doesn't exist, skipping..."
    fi
}

# Create test data for all users
create_test_files "amanafov"
create_test_files "testuser1"
create_test_files "testuser2"

# Create a shared public directory
if [ ! -d "/data/public" ]; then
    mkdir -p /data/public
    echo "Public shared directory - accessible to all" > /data/public/README.txt
    chown -R xrootd:xrootd /data/public
    chmod 755 /data/public
    chmod 644 /data/public/*
    echo "  [+] Created /data/public"
fi

echo ""
echo "============================================"
echo "Test Data Setup Complete!"
echo "============================================"
echo ""
echo "User Mapping Configuration:"
echo "  Token 'sub' claim    ->  Unix User    ->  Home Dir"
echo "  -------------------------------------------------"
echo "  a.manafov            ->  amanafov     ->  /data/amanafov"
echo "  testuser1            ->  testuser1    ->  /data/testuser1"
echo "  testuser2            ->  testuser2    ->  /data/testuser2"
echo "  <other>              ->  ACCESS DENIED (default_user=\"\")"
echo ""
echo "To test:"
echo "  1. Login with your GSI Keycloak account (a.manafov)"
echo "  2. Browse to /data - you should see all directories"
echo "  3. Try to access /data/amanafov - should work!"
echo "  4. Check XRootD logs for user mapping: docker logs dataharbor-xrootd-dev"
echo ""
