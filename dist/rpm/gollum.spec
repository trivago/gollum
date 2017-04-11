Name:           gollum
Version:        0.4.1
Release:        1%{?dist}
Summary:        Gollum message multiplexer
License:        Apache 2.0
URL:            https://github.com/trivago/gollum/releases/download/v%{version}/gollum-0.4.1.tar.gz
ExclusiveArch:  x86_64
Source0:        https://github.com/trivago/gollum/releases/download/v%{version}/gollum-0.4.1.tar.gz
Source1:        %{name}.sysconfig
Source2:        %{name}.service

BuildRequires:  golang >= 1.4.2
BuildRequires:  golang-pkg-linux-amd64
BuildRequires:  golang-pkg-bin-linux-amd64

Requires:       systemd


%description
Gollum is an n:m multiplexer that gathers messages from different sources and broadcasts them to a set of destinations.


%prep
%setup -q


%install
%{__install} -dm 755 %{buildroot}%{_sbindir}
%{__install} -m 755 %{name} %{buildroot}%{_sbindir}/%{name}

# install config files
%{__install} -dm 755 %{buildroot}%{_sysconfdir}/%{name}
%{__install} -m 644 config/*.conf %{buildroot}%{_sysconfdir}/%{name}/

# install sysconfig files
%{__install} -dm 755 %{buildroot}%{_sysconfdir}/sysconfig
%{__install} -m 644 %{S:1} %{buildroot}%{_sysconfdir}/sysconfig/%{name}

# install service files
%{__install} -dm 755 %{buildroot}%{_unitdir}
%{__install} -m 644 %{S:2} %{buildroot}%{_unitdir}/%{name}.service


%files
%{_sbindir}/%{name}
%{_unitdir}/%{name}.service
%dir %{_sysconfdir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}/*.conf
%config(noreplace) %{_sysconfdir}/sysconfig/%{name}


%pre
getent group gollum >/dev/null || groupadd -r gollum
getent passwd gollum >/dev/null || useradd -r -g gollum -d / -s /sbin/nologin \
        -c "Gollum multiplexer user" gollum


%post
%systemd_post gollum


%preun
%systemd_preun gollum


%postun
%systemd_postun


%clean
[ "%{buildroot}" != "/" ] && %{__rm} -rf %{buildroot}


%changelog
* Tue Dec 29 2015 Greg Markey <greg.markey@optiver.com.au> - 0.4.1-1
- Initial build.
