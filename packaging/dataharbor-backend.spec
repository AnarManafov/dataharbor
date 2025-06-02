Name:           dataharbor-backend
Version:        %{_version}
Release:        1%{?dist}
Summary:        DataHarbor Go Backend Application

License:        GPL-3.0
URL:            https://github.com/AnarManafov/dataharbor
Source0:        %{name}-%{version}.tar.gz

# The target architecture is provided via the command line using the --target option with the rpmbuild command
# BuildArch:      x86_64

%description
DataHarbor Go Backend Application.

%prep
# No preparation needed as we are using pre-built binaries

%build
# No build needed as we are using pre-built binaries

%install
mkdir -p %{buildroot}/usr/local/bin
install -m 0755 %{_sourcedir}/%{name} %{buildroot}/usr/local/bin/%{name}

%files
/usr/local/bin/%{name}

%changelog
