Vagrant.require_version ">= 1.4.0"

Vagrant.configure("2") do |config|
	config.vm.box = "precise64"
	config.vm.hostname = 'hystrix-go.local'

	config.vm.provision :shell, :path => "puppet/librarian.sh"
	config.vm.provision "puppet" do |puppet|
		puppet.manifests_path = "puppet/manifests"
		puppet.manifest_file = "go.pp"
	end

	config.vm.synced_folder ".", "/go/src/github.com/afex/hystrix-go"
end
