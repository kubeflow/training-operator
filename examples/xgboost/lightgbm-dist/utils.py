# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import re
import socket
import logging
import tempfile
from time import sleep
from typing import List, Union

logger = logging.getLogger(__name__)


def generate_machine_list_file(
    master_addr: str, master_port: str, worker_addrs: str, worker_port: str
) -> str:
    logger.info("starting to extract system env")

    filename = tempfile.NamedTemporaryFile(delete=False).name

    def _get_ips(
        master_addr_name,
        worker_addr_names,
        max_retries=10,
        sleep_secs=10,
        current_retry=0,
    ):
        try:
            worker_addr_ips = []
            master_addr_ip = socket.gethostbyname(master_addr_name)

            for addr in worker_addr_names.split(","):
                worker_addr_ips.append(socket.gethostbyname(addr))

        except socket.gaierror as ex:
            if "Name or service not known" in str(ex) and current_retry < max_retries:
                sleep(sleep_secs)
                master_addr_ip, worker_addr_ips = _get_ips(
                    master_addr_name,
                    worker_addr_names,
                    max_retries=max_retries,
                    sleep_secs=sleep_secs,
                    current_retry=current_retry + 1,
                )
            else:
                raise ValueError("Couldn't get address names")

        return master_addr_ip, worker_addr_ips

    master_ip, worker_ips = _get_ips(master_addr, worker_addrs)

    with open(filename, "w") as file:
        print(f"{master_ip} {master_port}", file=file)
        for addr in worker_ips:
            print(f"{addr} {worker_port}", file=file)

    return filename


def generate_train_conf_file(
    machine_list_file: str,
    world_size: int,
    output_model: str,
    local_port: Union[int, str],
    extra_args: List[str],
) -> str:

    filename = tempfile.NamedTemporaryFile(delete=False).name

    with open(filename, "w") as file:
        print("task = train", file=file)
        print(f"output_model = {output_model}", file=file)
        print(f"num_machines = {world_size}", file=file)
        print(f"local_listen_port = {local_port}", file=file)
        print(f"machine_list_file = {machine_list_file}", file=file)
        for arg in extra_args:
            m = re.match(r"--(.+)=([^\s]+)", arg)
            if m is not None:
                k, v = m.groups()
                print(f"{k} = {v}", file=file)

    return filename
