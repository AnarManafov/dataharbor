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

%description
DataHarbor Go Backend Application - statically linked for maximum compatibility.
This version is built with CGO_ENABLED=0 for static linking, eliminating GLIBC dependencies.

%prep
# No preparation needed as we are using pre-built binaries

%build
# No build needed as we are using pre-built binaries

%install
mkdir -p %{buildroot}/usr/local/bin
install -m 0755 %{_sourcedir}/%{name} %{buildroot}/usr/local/bin/%{name}

# Add architecture information to the installed binary
mkdir -p %{buildroot}/usr/local/share/dataharbor
echo "Architecture: %{_target_cpu}" > %{buildroot}/usr/local/share/dataharbor/arch-info.txt
echo "Build type: Static linking (CGO_ENABLED=0)" >> %{buildroot}/usr/local/share/dataharbor/arch-info.txt

%files
/usr/local/bin/%{name}
/usr/local/share/dataharbor/arch-info.txt

%changelog
