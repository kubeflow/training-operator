import torch

torch.distributed.init_process_group(init_method="env://")
rank = torch.distributed.get_rank()
world_size = torch.distributed.get_world_size()
print(f"rank {rank} world_size {world_size}")
a = torch.tensor([1])
torch.distributed.all_reduce(a)
print(f"rank {rank} world_size {world_size} result {a}")
torch.distributed.barrier()
print(f"rank {rank} world_size {world_size}")
