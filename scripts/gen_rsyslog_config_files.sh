#! /bin/bash

set -x

mkdir files/rsyslog/roles
LOCAL_FILE_OWNER=$( id -un )
LOCAL_FILE_GROUP=$( id -gn )
WORKCONFDIR=$( mktemp -d )
sudo ansible-playbook -vvv -e@files/rsyslog/vars.yaml -e logging_enabled=true \
    -e logging_role_path="../../../vendor/github.com/linux-system-roles/logging" \
    -e rsyslog_file_group=${LOCAL_FILE_GROUP} -e rsyslog_file_owner=${LOCAL_FILE_OWNER} \
	-e rsyslog_parent_config_dir=${WORKCONFDIR} \
	-e rsyslog_file_config_dir=${WORKCONFDIR} \
	-e rsyslog_mode=0664 \
    files/rsyslog/playbook.yaml > /tmp/rsyslog.ansible.out 2>&1
if [ $? -ne 0 ]; then
    echo "Generating rsyslog files from linux-system-roles/logging failed."
    exit 1
fi
mv --force ${WORKCONFDIR}/* files/rsyslog
rmdir ${WORKCONFDIR}
rmdir files/rsyslog/roles

