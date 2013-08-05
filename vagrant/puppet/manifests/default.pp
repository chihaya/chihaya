Exec { path => '/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin/' }
exec { 'echo this works': }

group { 'puppet': ensure => 'present' }

exec { 'chown -R vagrant:vagrant /home/vagrant/':
}

exec { 'apt-get update':
	command => '/usr/bin/apt-get update',
}

exec { 'add-apt-repository ppa:chris-lea/zeromq && apt-get update':
	require => Package['python-software-properties'],
	alias   => 'zmq_repo',
	creates => '/etc/apt/sources.list.d/chris-lea-zeromq-precise.list',
}

exec { 'add-apt-repository ppa:duh/golang && apt-get update':
	alias   => 'go_repo',
	creates => '/etc/apt/sources.list.d/gophers-go-precise.list',
	require => Package['python-software-properties'],
}

package { 'pkg-config':
	require => Exec['apt-get update'],
	ensure  => present,
}

package { 'libzmq-dev':
	require => [
		Exec['zmq_repo'],
		Package['pkg-config'],
	],
	ensure  => present,
}

package { 'python-software-properties':
	require => Exec['apt-get update'],
	ensure  => present,
}

package { 'git':
	require => Exec['apt-get update'],
	ensure  => present,
}

package { 'golang':
	require => Exec['go_repo'],
	ensure  => present,
}

exec { 'echo "export GOPATH=/home/vagrant/chihaya" > /etc/profile.d/gopath.sh':
	alias   => 'go_path',
	creates => '/etc/profile.d/gopath.sh',
}
