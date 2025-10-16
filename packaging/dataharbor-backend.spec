Name:           dataharbor-backend
Version:        %{_version}
Release:        1%{?dist}
Summary:        DataHarbor Go Backend Application

License:        GPL-3.0
URL:            https://github.com/AnarManafov/dataharbor
Source0:        %{name}-%{version}.tar.gz

# Automatically determine architecture from the build environment
# The target architecture is provided via the command line using the --target option with the rpmbuild command
# BuildArch:      %{_target_cpu}

# Require systemd for service management
Requires:       systemd
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd

%description
DataHarbor Go Backend Application - statically linked for maximum compatibility.
This version is built with CGO_ENABLED=0 for static linking, eliminating GLIBC dependencies.
Includes systemd service file and example configuration.

%prep
# No preparation needed as we are using pre-built binaries

%build
# No build needed as we are using pre-built binaries

%install
# Install binary
mkdir -p %{buildroot}/usr/local/bin
install -m 0755 %{_sourcedir}/%{name} %{buildroot}/usr/local/bin/%{name}

# Install systemd service file
mkdir -p %{buildroot}/usr/lib/systemd/system
install -m 0644 %{_sourcedir}/../systemd/dataharbor-backend.service %{buildroot}/usr/lib/systemd/system/dataharbor-backend.service

# Create configuration directory
mkdir -p %{buildroot}/etc/dataharbor

# Install example configuration
install -m 0644 %{_sourcedir}/../config/application.yaml.example %{buildroot}/etc/dataharbor/application.yaml.example

# Create log directory
mkdir -p %{buildroot}/var/log/dataharbor

# Install documentation
mkdir -p %{buildroot}/usr/share/doc/dataharbor-backend
echo "DataHarbor Backend Service" > %{buildroot}/usr/share/doc/dataharbor-backend/README.md
echo "" >> %{buildroot}/usr/share/doc/dataharbor-backend/README.md
echo "Configuration: /etc/dataharbor/application.yaml" >> %{buildroot}/usr/share/doc/dataharbor-backend/README.md
echo "Service: systemctl status dataharbor-backend" >> %{buildroot}/usr/share/doc/dataharbor-backend/README.md
echo "Logs: journalctl -u dataharbor-backend" >> %{buildroot}/usr/share/doc/dataharbor-backend/README.md
echo "Documentation: https://github.com/AnarManafov/dataharbor/tree/master/docs" >> %{buildroot}/usr/share/doc/dataharbor-backend/README.md

# Add architecture information
mkdir -p %{buildroot}/usr/local/share/dataharbor
echo "Architecture: %{_target_cpu}" > %{buildroot}/usr/local/share/dataharbor/arch-info.txt
echo "Build type: Static linking (CGO_ENABLED=0)" >> %{buildroot}/usr/local/share/dataharbor/arch-info.txt

%files
/usr/local/bin/%{name}
/usr/lib/systemd/system/dataharbor-backend.service
/etc/dataharbor/application.yaml.example
%dir /etc/dataharbor
%dir /var/log/dataharbor
/usr/share/doc/dataharbor-backend/README.md
/usr/local/share/dataharbor/arch-info.txt

%post
# Post-installation script
systemctl daemon-reload >/dev/null 2>&1 || :

# Display installation message
cat << 'EOF'

╔════════════════════════════════════════════════════════════════════════════╗
║                                                                            ║
║  DataHarbor Backend installed successfully!                                ║
║                                                                            ║
╚════════════════════════════════════════════════════════════════════════════╝

Installation Summary:
   • Binary: /usr/local/bin/dataharbor-backend
   • SystemD Service: /usr/lib/systemd/system/dataharbor-backend.service
   • Config Example: /etc/dataharbor/application.yaml.example
   • Log Directory: /var/log/dataharbor/

Next Steps:

   1. Create configuration file:
      sudo cp /etc/dataharbor/application.yaml.example /etc/dataharbor/application.yaml
      sudo nano /etc/dataharbor/application.yaml

   2. Update the following in your config:
      • OIDC client_secret (get from your OIDC provider)
      • OIDC session_secret (generate with: openssl rand -base64 32)
      • Frontend URL (your actual domain)
      • SSL certificate paths
      • XRootD server settings

   3. Enable and start the service:
      sudo systemctl enable dataharbor-backend
      sudo systemctl start dataharbor-backend

   4. Check service status:
      sudo systemctl status dataharbor-backend

   5. View logs:
      sudo journalctl -u dataharbor-backend -f

   6. Test health endpoint:
      curl -k https://localhost:8081/health

Documentation: /usr/share/doc/dataharbor-backend/README.md
Online Docs: https://github.com/AnarManafov/dataharbor/tree/master/docs

EOF

%preun
# Pre-uninstallation script
if [ $1 -eq 0 ]; then
    # Package removal, not upgrade
    systemctl --no-reload disable dataharbor-backend.service >/dev/null 2>&1 || :
    systemctl stop dataharbor-backend.service >/dev/null 2>&1 || :
fi

%postun
# Post-uninstallation script
systemctl daemon-reload >/dev/null 2>&1 || :

%changelog
