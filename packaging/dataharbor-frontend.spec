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

%prep
# No preparation needed as we are using pre-built binaries

%build
# No build needed as we are using pre-built binaries

%install
mkdir -p %{buildroot}/usr/share/%{name}
cp -r %{_sourcedir}/%{name}-%{version}/* %{buildroot}/usr/share/%{name}/

# Install nginx configuration
mkdir -p %{buildroot}/etc/dataharbor-frontend/nginx
install -m 0644 %{_sourcedir}/nginx.conf %{buildroot}/etc/dataharbor-frontend/nginx/nginx.conf


%files
/usr/share/%{name}
/etc/dataharbor-frontend/nginx/nginx.conf

%changelog
