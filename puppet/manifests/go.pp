class { 'golang':
  version   => '1.2',
  workspace => '/usr/local/src/go',
}

file { '/home/vagrant/hystrix-go':
   ensure => 'link',
   target => '/usr/local/src/go/src/hystrix-go',
}