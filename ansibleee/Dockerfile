FROM quay.io/ansible/ansible-runner:latest
USER root
RUN chmod g=u /etc/passwd /etc/group
# Install tripleo-ansible content
# Install packages from tripleo-ansible bindep
RUN dnf -y install gcc-c++  git libffi-devel openssl-devel podman \
    python3-devel python3-pyyaml python3-dnf python-rhsm-certificates python3-libselinux python3-libsemanage \
    gzip gettext && dnf clean all && rm -rf /var/cache/{dnf,yum} && rm -rf /var/lib/dnf/history.* && rm -rf /var/log/*
RUN /usr/bin/git clone https://github.com/openstack/tripleo-ansible /var/tmp/tripleo-ansible
RUN cd /var/tmp/tripleo-ansible && pip install -r requirements.txt
# ansible-galaxy issue with PyOpenSSL https://github.com/ansible/awx/issues/12124
RUN pip3 install 'pyOpenSSL<20.0.0' 'cryptography >35,<37'
RUN cd /var/tmp/tripleo-ansible && ansible-galaxy role install -r requirements.yml --roles-path "/usr/share/ansible/roles" && \
    ansible-galaxy collection install -r requirements.yml --collections-path "/usr/share/ansible/collections"
RUN cd /var/tmp/tripleo-ansible && python3 setup.py install --prefix=/usr
RUN rm -fr /var/tmp/tripleo-ansible
RUN chmod -R 777 /usr/share/ansible
COPY settings /runner/env/settings
RUN chmod 777 /runner/env/settings
COPY tripleo_entrypoint.sh /bin/tripleo_entrypoint
RUN sed -i '1d' /bin/entrypoint
RUN cat /bin/entrypoint >> /bin/tripleo_entrypoint
RUN chmod +x /bin/tripleo_entrypoint
WORKDIR /runner
RUN rm -rf roles && ln -snf /usr/share/ansible/roles roles
RUN rm -rf project && ln -snf /usr/share/ansible/tripleo-playbooks project
USER 1001
ENTRYPOINT ["tripleo_entrypoint"]
