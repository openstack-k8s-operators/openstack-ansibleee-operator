# Persistent logs

For enabling persistent logging, you need to mount `/runner/artifacts` into a [persistent Volume](https://kubernetes.io/docs/concepts/storage/volumes/) through `extraMounts` field.

#### Example:
```yaml
apiVersion: ansibleee.openstack.org/v1alpha1
kind: OpenStackAnsibleEE
metadata:
  name: ansibleee-logs
  namespace: openstack
spec:
  playbook: "test.yaml"
  image: quay.io/openstack-k8s-operators/openstack-ansibleee-runner:latest
  inventory: |
          all:
            hosts:
              localhost
  extraMounts:
    - extraVolType: Logs
      volumes:
      - name: ansible-logs
        persistentVolumeClaim:
          claimName: openstack-ansible-logs
      mounts:
      - name: ansible-logs
        mountPath: "/runner/artifacts"
```
