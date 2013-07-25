
from __future__ import unicode_literals

from api.models import Node
from celery import task


@task(name='mock.build_layer')
def build_layer(layer, creds, params):
    # create security group any other infrastructure
    return


@task(name='mock.destroy_layer')
def destroy_layer(layer, creds, params):
    # delete security group and any other infrastructure
    return


@task(name='mock.launch_node')
def launch_node(node_id, creds, params, init, ssh_username, ssh_private_key):
    node = Node.objects.get(uuid=node_id)
    node.provider_id = 'i-1234567'
    node.metadata = {'state': 'running'}
    node.fqdn = 'localhost.localdomain.local'
    node.save()


@task(name='mock.terminate_node')
def terminate_node(node_id, creds, params, provider_id):
    node = Node.objects.get(uuid=node_id)
    node.metadata = {'state': 'terminated'}
    node.save()
    # delete the node itself from the database
    node.delete()


@task(name='mock.converge_node')
def converge_node(node_id, ssh_username, fqdn, ssh_private_key):
    output = ""
    rc = 0
    return output, rc
