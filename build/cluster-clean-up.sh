#!/bin/bash
# Copyright Red Hat



function hub() {
    echo "Hub: clean up"
#    oc delete ns duplicatetest || true
#    oc delete ns -l e2e=true || true
}

function managed() {
    echo "Managed: clean up"
#    oc delete ns -l e2e=true || true
}

case $1 in
    hub)
        hub
        ;;
    managed)
        managed
        ;;
    *)
        hub
        managed
        ;;
esac
