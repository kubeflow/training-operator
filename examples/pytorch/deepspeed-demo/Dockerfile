FROM deepspeed/deepspeed:v072_torch112_cu117

RUN apt update
RUN apt install -y ninja-build

WORKDIR /
COPY requirements.txt .
COPY train_bert_ds.py .

RUN pip install -r requirements.txt
RUN mkdir -p /root/deepspeed_data
