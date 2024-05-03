#!/usr/bin/env python3
import io
import os
import pprint
import sys
import time

import torch.distributed as dist

if __name__ == "__main__":

    env_dict = {
        k: os.environ[k]
        for k in (
            "LOCAL_RANK",
            "RANK",
            "GROUP_RANK",
            "WORLD_SIZE",
            "MASTER_ADDR",
            "MASTER_PORT",
            "TORCHELASTIC_RESTART_COUNT",
            "TORCHELASTIC_MAX_RESTARTS",
        )
    }

    with io.StringIO() as buff:
        print("======================================================", file=buff)
        print(
            f"Environment variables set by the agent on PID {os.getpid()}:", file=buff
        )
        pprint.pprint(env_dict, stream=buff)
        print("======================================================", file=buff)
        print(buff.getvalue())
        sys.stdout.flush()

    dist.init_process_group(backend="gloo")
    dist.barrier()

    print(
        (
            f"On PID {os.getpid()}, after init process group, "
            f"rank={dist.get_rank()}, world_size = {dist.get_world_size()}\n"
        )
    )
