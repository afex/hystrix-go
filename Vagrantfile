Vagrant.require_version ">= 1.4.0"

Vagrant.configure("2") do |config|
	config.vm.box = "hystrix-go"
	config.vm.box_url = "http://files.vagrantup.com/precise64.box"
	config.vm.hostname = 'hystrix-go.local'

	config.vm.provision :shell, :path => "puppet/librarian_preinstall.sh"
	config.vm.provision "puppet" do |puppet|
		puppet.manifests_path = "puppet/manifests"
		puppet.manifest_file = "go.pp"
	end

	config.vm.synced_folder ".", "/usr/local/src/go/src/hystrix-go"
end