# Copyright Red Hat
#!/usr/bin/env bash

# echo "Install awx cli tool"
command -v awx &> /dev/null
if [[ $? -ne 0 ]]; then
    wget -O ansible-tower-cli-3.8.3-1.tar.gz https://releases.ansible.com/ansible-tower/cli/ansible-tower-cli-3.8.3-1.tar.gz
    tar zxvf ansible-tower-cli-3.8.3-1.tar.gz 
    pushd awxkit-3.8.3
    python3 setup.py install
    popd
fi

echo "Set up ansible project and job templates for CLC automation test..."
$(awx -k login -f human TOWER_USERNAME=${TOWER_USER} TOWER_PASSWORD=${TOWER_PASSWORD})
awx config

# delete existing project+job_template
if [[ $(awx -k project list --name $1 | jq .count) > 0 ]]; then 
    echo 'Ansible project found - Deleting...'
    awx -k job_template delete --name $2
    awx -k projects delete --name $1
else 
    echo 'Ansible project not found'
fi

# add project+job_template
echo 'Creating Ansible project+job_template for CLC testing...'
awx -k projects create --wait \
    --organization 1 --name=$1 \
    --scm_type git --scm_url $3 \
    -f human
awx -k job_templates create \
    --name=$2 --project $1 \
    --playbook hello_world.yml --inventory 'Demo Inventory' \
    --ask_variables_on_launch true \
    --ask_inventory_on_launch true \
    -f human

# update options.yaml for ansible connections
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
SED='sed'
if [[ "${OS}" == "darwin" ]]; then
    SED='gsed'
fi
${SED} -i "s|^\(\s*ansibleHost\s*:\s*\).*|\1${TOWER_HOST}|" options.yaml
${SED} -i "s|^\(\s*ansibleToken\s*:\s*\).*|\1${TOWER_OAUTH_TOKEN}|" options.yaml