Exec { path => '/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin/' }
exec { 'echo this works': }

group { 'puppet': ensure => 'present' }

exec { 'chown -R vagrant:vagrant /home/vagrant/': }

exec { 'apt-get update':
	command => '/usr/bin/apt-get update',
}

exec { 'add-apt-repository ppa:duh/golang && apt-get update':
	alias   => 'go_repo',
	creates => '/etc/apt/sources.list.d/gophers-go-precise.list',
	require => Package['python-software-properties'],
}

package { 'python-software-properties':
	ensure  => present,
	require => Exec['apt-get update'],
}

package { 'git':
	ensure  => present,
	require => Exec['apt-get update'],
}

package { 'golang':
	ensure  => present,
	require => Exec['go_repo'],
}

exec { 'echo "export GOPATH=/home/vagrant/chihaya" > /etc/profile.d/gopath.sh':
	creates => '/etc/profile.d/gopath.sh',
}
