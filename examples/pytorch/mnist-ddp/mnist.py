import argparse

import torch
from kubeflow.training import Trainer, TrainingClient


def train_func(dict):

    import os

    import torch
    import torch.distributed as dist
    import torch.nn as nn
    import torch.nn.functional as F
    import torchvision.transforms as transforms
    from torch.nn.parallel import DistributedDataParallel
    from torch.optim.lr_scheduler import StepLR
    from torch.utils.data import DataLoader
    from torch.utils.data.distributed import DistributedSampler
    from torchvision.datasets import MNIST

    backend = dict.get("backend")
    batch_size = dict.get("batch_size")
    test_batch_size = dict.get("test_batch_size")
    epochs = dict.get("epochs")
    lr = dict.get("lr")
    lr_gamma = dict.get("lr_gamma")
    lr_period = dict.get("lr_period")
    seed = dict.get("seed")
    log_interval = dict.get("log_interval")
    save_model = dict.get("save_model")

    class Net(nn.Module):
        def __init__(self):
            super(Net, self).__init__()
            self.conv1 = nn.Conv2d(1, 32, 3, 1)
            self.conv2 = nn.Conv2d(32, 64, 3, 1)
            self.dropout1 = nn.Dropout(0.25)
            self.dropout2 = nn.Dropout(0.5)
            self.fc1 = nn.Linear(9216, 128)
            self.fc2 = nn.Linear(128, 10)

        def forward(self, x):
            x = self.conv1(x)
            x = F.relu(x)
            x = self.conv2(x)
            x = F.relu(x)
            x = F.max_pool2d(x, 2)
            x = self.dropout1(x)
            x = torch.flatten(x, 1)
            x = self.fc1(x)
            x = F.relu(x)
            x = self.dropout2(x)
            x = self.fc2(x)
            output = F.log_softmax(x, dim=1)
            return output

    def train(model, device, criterion, train_loader, optimizer, epoch, log_interval):
        model.train()
        for batch_idx, (data, target) in enumerate(train_loader):
            data, target = data.to(device), target.to(device)
            optimizer.zero_grad()
            output = model(data)
            loss = criterion(output, target)
            loss.backward()
            optimizer.step()
            if batch_idx % log_interval == 0:
                print(
                    "Train Epoch: {} [{}/{} ({:.0f}%)]\tLoss: {:.6f}".format(
                        epoch,
                        batch_idx * len(data),
                        len(train_loader.dataset),
                        100.0 * batch_idx / len(train_loader),
                        loss.item(),
                    )
                )

    def test(model, device, criterion, rank, test_loader, epoch):
        model.eval()
        samples = 0
        local_loss = 0
        local_correct = 0
        with torch.no_grad():
            for data, target in test_loader:
                samples += len(data)
                data, target = data.to(device), target.to(device)
                output = model(data)
                # sum up batch loss
                local_loss += criterion(output, target).item()
                # get the index of the max log-probability
                pred = output.argmax(dim=1, keepdim=True)
                local_correct += pred.eq(target.view_as(pred)).sum().item()

        local_accuracy = 100.0 * local_correct / samples

        header = f"{'-'*15} Epoch {epoch} Evaluation {'-'*15}"
        print(f"\n{header}\n")
        # Log local metrics on each rank
        print(f"Local rank {rank}:")
        print(f"- Loss: {local_loss / samples:.4f}")
        print(f"- Accuracy: {local_correct}/{samples} ({local_accuracy:.0f}%)\n")

        # To Tensors so local metrics can be globally reduced across ranks
        global_loss = torch.tensor([local_loss], device=device)
        global_correct = torch.tensor([local_correct], device=device)

        # Reduce the metrics on rank 0
        dist.reduce(global_loss, dst=0, op=torch.distributed.ReduceOp.SUM)
        dist.reduce(global_correct, dst=0, op=torch.distributed.ReduceOp.SUM)

        # Log the aggregated metrics only on rank 0
        if rank == 0:
            global_loss = global_loss / len(test_loader.dataset)
            global_accuracy = (global_correct.double() / len(test_loader.dataset)) * 100
            global_correct = global_correct.int().item()
            samples = len(test_loader.dataset)
            print("Global metrics:")
            print(f"\t - Loss: {global_loss.item():.6f}")
            print(
                f"\t - Accuracy: {global_correct}/{samples} ({global_accuracy.item():.2f}%)"
            )
        else:
            print("See rank 0 logs for global metrics")
        print(f"\n{'-'*len(header)}\n")

    dist.init_process_group(backend=backend)

    torch.manual_seed(seed)
    rank = int(os.environ["RANK"])
    local_rank = int(os.environ["LOCAL_RANK"])

    model = Net()

    use_cuda = torch.cuda.is_available()
    if use_cuda:
        if backend != torch.distributed.Backend.NCCL:
            print(
                "Please use NCCL distributed backend for the best performance using NVIDIA GPUs"
            )
        device = torch.device(f"cuda:{local_rank}")
        model = DistributedDataParallel(model.to(device), device_ids=[local_rank])
    else:
        device = torch.device("cpu")
        model = DistributedDataParallel(model.to(device))

    transform = transforms.Compose(
        [
            transforms.ToTensor(),
            transforms.Normalize((0.1307,), (0.3081,)),
        ]
    )

    train_dataset = MNIST(root="./data", train=True, transform=transform, download=True)
    train_sampler = DistributedSampler(train_dataset)
    train_loader = DataLoader(
        dataset=train_dataset,
        batch_size=batch_size,
        sampler=train_sampler,
        pin_memory=use_cuda,
    )

    test_dataset = MNIST(root="./data", train=False, transform=transform, download=True)
    test_sampler = DistributedSampler(test_dataset)
    test_loader = DataLoader(
        dataset=test_dataset,
        batch_size=test_batch_size,
        sampler=test_sampler,
        pin_memory=use_cuda,
    )

    criterion = nn.CrossEntropyLoss().to(device)
    optimizer = torch.optim.SGD(model.parameters(), lr=lr)
    scheduler = StepLR(optimizer, step_size=lr_period, gamma=lr_gamma)

    for epoch in range(1, epochs + 1):
        train(model, device, criterion, train_loader, optimizer, epoch, log_interval)
        test(model, device, criterion, rank, test_loader, epoch)
        scheduler.step()

    if save_model:
        torch.save(model.state_dict(), "mnist.pt")

    # Wait so rank 0 can gather the global metrics
    dist.barrier()
    dist.destroy_process_group()


parser = argparse.ArgumentParser(description="PyTorch DDP MNIST Training Example")

parser.add_argument(
    "--batch-size",
    type=int,
    default=100,
    metavar="N",
    help="input batch size for training [100]",
)

parser.add_argument(
    "--test-batch-size",
    type=int,
    default=100,
    metavar="N",
    help="input batch size for testing [100]",
)

parser.add_argument(
    "--epochs",
    type=int,
    default=10,
    metavar="N",
    help="number of epochs to train [10]",
)

parser.add_argument(
    "--lr",
    type=float,
    default=1e-1,
    metavar="LR",
    help="learning rate [1e-1]",
)

parser.add_argument(
    "--lr-gamma",
    type=float,
    default=0.5,
    metavar="G",
    help="learning rate decay factor [0.5]",
)

parser.add_argument(
    "--lr-period",
    type=float,
    default=20,
    metavar="P",
    help="learning rate decay period in step size [20]",
)

parser.add_argument(
    "--seed",
    type=int,
    default=0,
    metavar="S",
    help="random seed [0]",
)

parser.add_argument(
    "--log-interval",
    type=int,
    default=10,
    metavar="N",
    help="how many batches to wait before logging training metrics [10]",
)

parser.add_argument(
    "--save-model",
    action="store_true",
    default=False,
    help="saving the trained model [False]",
)

parser.add_argument(
    "--backend",
    type=str,
    choices=[torch.distributed.Backend.GLOO, torch.distributed.Backend.NCCL],
    default=torch.distributed.Backend.NCCL,
    help="Distributed backend [NCCL]",
)

parser.add_argument(
    "--num-workers",
    type=int,
    default=1,
    metavar="N",
    help="Number of workers [1]",
)

parser.add_argument(
    "--worker-resources",
    type=str,
    nargs=2,
    action="append",
    dest="resources",
    default=[
        ("cpu", 1),
        ("memory", "2Gi"),
        ("nvidia.com/gpu", 1),
    ],
    metavar=("RESOURCE", "QUANTITY"),
    help="Resources per worker [cpu: 1, memory: 2Gi, nvidia.com/gpu: 1]",
)

parser.add_argument(
    "--runtime",
    type=str,
    default="torch-distributed",
    metavar="NAME",
    help="the training runtime [torch-distributed]",
)

args = parser.parse_args()

client = TrainingClient()

job_name = client.train(
    runtime_ref=args.runtime,
    trainer=Trainer(
        func=train_func,
        func_args={
            "backend": args.backend,
            "batch_size": args.batch_size,
            "test_batch_size": args.test_batch_size,
            "epochs": args.epochs,
            "lr": args.lr,
            "lr_gamma": args.lr_gamma,
            "lr_period": args.lr_period,
            "seed": args.seed,
            "log_interval": args.log_interval,
            "save_model": args.save_model,
        },
        num_nodes=args.num_workers,
        resources_per_node={
            resource: quantity for (resource, quantity) in args.resources
        },
    ),
)

client.get_job_logs(job_name, follow=True)
