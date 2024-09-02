incus launch images:debian/bookworm composetest --config security.nesting=true --config security.privileged=true

incus file push ./installincus.sh composetest/root/installincus.sh

incus exec composetest -- bash /root/installincus.sh