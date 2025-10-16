Name:           dataharbor-frontend
Version:        %{_version}
Release:        1%{?dist}
Summary:        DataHarbor Vue.js Frontend Application

License:        GPL-3.0
URL:            https://github.com/AnarManafov/dataharbor
Source0:        %{name}-%{version}.tar.gz

BuildArch:      noarch
Requires:       nginx

%description
DataHarbor Vue.js Frontend Application.
Includes multiple nginx configuration templates for different deployment scenarios.

%prep
# No preparation needed as we are using pre-built binaries

%build
# No build needed as we are using pre-built binaries

%install
# Install frontend files
mkdir -p %{buildroot}/usr/share/%{name}
cp -r %{_sourcedir}/%{name}-%{version}/* %{buildroot}/usr/share/%{name}/

# Install example frontend configuration
install -m 0644 %{_sourcedir}/../config/config.json.example %{buildroot}/usr/share/%{name}/config.json.example

# Install nginx configuration templates
mkdir -p %{buildroot}/etc/dataharbor-frontend/nginx/templates
install -m 0644 %{_sourcedir}/../nginx/nginx-http-simple.conf %{buildroot}/etc/dataharbor-frontend/nginx/templates/nginx-http-simple.conf
install -m 0644 %{_sourcedir}/../nginx/nginx-https-proxy.conf %{buildroot}/etc/dataharbor-frontend/nginx/templates/nginx-https-proxy.conf
install -m 0644 %{_sourcedir}/../nginx/nginx-gsi.conf %{buildroot}/etc/dataharbor-frontend/nginx/templates/nginx-gsi.conf

# Keep backward compatibility - install simple HTTP config as default
install -m 0644 %{_sourcedir}/../nginx/nginx-http-simple.conf %{buildroot}/etc/dataharbor-frontend/nginx/nginx.conf


%files
/usr/share/%{name}
/etc/dataharbor-frontend/nginx/nginx.conf
/etc/dataharbor-frontend/nginx/templates/nginx-http-simple.conf
/etc/dataharbor-frontend/nginx/templates/nginx-https-proxy.conf
/etc/dataharbor-frontend/nginx/templates/nginx-gsi.conf

%post
# Post-installation script
cat << 'EOF'

╔════════════════════════════════════════════════════════════════════════════╗
║                                                                            ║
║  DataHarbor Frontend installed successfully!                               ║
║                                                                            ║
╚════════════════════════════════════════════════════════════════════════════╝

Installation Summary:
   • Frontend Files: /usr/share/dataharbor-frontend/
   • Config Example: /usr/share/dataharbor-frontend/config.json.example
   • Nginx Templates: /etc/dataharbor-frontend/nginx/templates/

Available Nginx Configuration Templates:

   1. Simple HTTP (Development/Testing):
      /etc/dataharbor-frontend/nginx/templates/nginx-http-simple.conf
      → Basic HTTP on port 80, no SSL, no reverse proxy
      → Frontend accesses backend directly

   2. HTTPS with Reverse Proxy (Recommended for Production):
      /etc/dataharbor-frontend/nginx/templates/nginx-https-proxy.conf
      → HTTPS on port 443, SSL enabled, reverse proxy to backend
      → All traffic goes through nginx

   3. GSI-Specific Configuration:
      /etc/dataharbor-frontend/nginx/templates/nginx-gsi.conf
      → HTTPS on port 443 (XRootD uses port 80)
      → Backend on port 22000, SSL with GEANT CA certificates

Next Steps:

   1. Choose and copy a nginx template:
      
      For production with HTTPS:
      sudo cp /etc/dataharbor-frontend/nginx/templates/nginx-https-proxy.conf \
              /etc/nginx/conf.d/dataharbor.conf
      
      For GSI deployment:
      sudo cp /etc/dataharbor-frontend/nginx/templates/nginx-gsi.conf \
              /etc/nginx/conf.d/dataharbor.conf
      
      For simple HTTP (testing only):
      sudo cp /etc/dataharbor-frontend/nginx/templates/nginx-http-simple.conf \
              /etc/nginx/conf.d/dataharbor.conf

   2. Edit the nginx config for your environment:
      sudo nano /etc/nginx/conf.d/dataharbor.conf
      
      Update:
      • server_name (your actual hostname/domain)
      • SSL certificate paths
      • Backend proxy_pass URL and port

   3. Create frontend configuration:
      sudo cp /usr/share/dataharbor-frontend/config.json.example \
              /usr/share/dataharbor-frontend/config.json
      sudo nano /usr/share/dataharbor-frontend/config.json
      
      Update:
      • apiBaseUrl ("/api" for reverse proxy, or direct backend URL)
      • OIDC settings (authority, client_id, redirect_uri)

   4. Test nginx configuration:
      sudo nginx -t

   5. Reload nginx:
      sudo systemctl reload nginx

   6. Test in browser:
      https://your-hostname/

Configuration Tips:

   • If using reverse proxy: apiBaseUrl = "/api"
   • If direct backend access: apiBaseUrl = "https://backend:8081/api"
   • For GSI: Use port 443 for frontend, 22000 for backend
   • Comment out default nginx server block if it conflicts with port 80

Documentation: https://github.com/AnarManafov/dataharbor/tree/master/docs

EOF

%changelog
