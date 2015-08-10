Vagrant.configure("2") do |config|
	config.vm.box = "ubuntu/trusty64"
	config.vm.hostname = 'hystrix-go.local'

	config.vm.provision :shell, :path => "scripts/vagrant.sh"
	
	config.vm.synced_folder ".", "/go/src/github.com/afex/hystrix-go"

	config.vm.provider "virtualbox" do |v|
		v.cpus = 3
	end
end
