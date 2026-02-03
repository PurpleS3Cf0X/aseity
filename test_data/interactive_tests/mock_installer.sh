#!/bin/bash
# mock_installer.sh
# Simulates a complex installation requiring permissions

echo "--- Aseity Fake Installer v1.0 ---"
echo "Checking dependencies..."
sleep 1
echo "Dependencies OK."

echo "We need root privileges to install to /opt/fake-app."
echo -n "Password: " # -n to avoid newline, mimic sudo
read -s password # -s for silent input
echo

if [ "$password" == "secret123" ]; then
    echo "Authentication successful!"
    echo "Installing..."
    sleep 1
    echo "Installation Complete!"
else
    echo "Authentication failed. Access denied."
    exit 1
fi
