import logging
import os
import json
import torch
import torch.distributed as dist
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim

from math import ceil
from random import Random
from torch.autograd import Variable
from torchvision import datasets, transforms

def run():
    """ Simple Send/Recv for testing Master <--> Workers communication """
    rank = dist.get_rank()
    size = dist.get_world_size()
    inp = torch.randn(2,2)
    result = torch.zeros(2,2)
    if rank == 0:
        # Send the input tensor to all workers
        for i in range(1, size):
            dist.send(tensor=inp, dst=i)
            # Receive the result tensor from all workers
            dist.recv(tensor=result, src=i)
            logging.info("Result from worker %d : %s", i, result)
    else:
        # Receive input tensor from master
        dist.recv(tensor=inp, src=0)
        # Elementwise tensor multiplication
        result = torch.mul(inp,inp)
        # Send the result tensor back to master
        dist.send(tensor=result, dst=0)

def init_processes(fn, backend='gloo'):
    """ Initialize the distributed environment. """
    dist.init_process_group(backend)
    fn()

def main():
    logging.info("Torch version: %s", torch.__version__)
    
    port = os.environ.get("MASTER_PORT", "{}")
    logging.info("MASTER_PORT: %s", port)
    
    addr = os.environ.get("MASTER_ADDR", "{}")
    logging.info("MASTER_ADDR: %s", addr)

    world_size = os.environ.get("WORLD_SIZE", "{}")
    logging.info("WORLD_SIZE: %s", world_size)
    
    rank = os.environ.get("RANK", "{}")
    logging.info("RANK: %s", rank)
    
    init_processes(run)


if __name__ == "__main__":
    logging.getLogger().setLevel(logging.INFO)
    main()
