Name:           data-lake-ui-frontend
Version:        0.6.0
Release:        1%{?dist}
Summary:        data-lake-ui Vue.js Frontend Application

License:        GPL-3.0
URL:            https://github.com/AnarManafov/data-lake-ui
Source0:        %{name}-%{version}.tar.gz

BuildArch:      noarch

%description
data-lake-ui Vue.js Frontend Application.

%prep
# No preparation needed as we are using pre-built binaries

%build
# No build needed as we are using pre-built binaries

%install
mkdir -p %{buildroot}/usr/share/%{name}
cp -r %{_sourcedir}/%{name}-%{version}/* %{buildroot}/usr/share/%{name}/

%files
/usr/share/%{name}

%changelog
