
BOOT_ADDRESS=0x50543e830590fD03a0301fAA0164d731f0E2ff7D
NODE_ADDRESS=0xF75E32Cf73De1FF96fc080Ffc603b22CaFC0Ba8F

BOOT_DATA_DIR=tbb-data
NODE_DATA_DIR=tbb-data-01

BOOT_IP=127.0.0.1
BOOT_PORT=8111

NODE_IP=127.0.0.1
NODE_PORT=8112

bootstrap:
	.\tbb.exe run \
		--datadir=$(BOOT_DATA_DIR) \
		--ip=$(BOOT_IP) \
		--port=$(BOOT_PORT) \
		--bootstrap-ip=$(BOOT_IP) \
		--bootstrap-port=$(BOOT_PORT) \
		--disable-ssl \
		--miner=$(BOOT_ADDRESS)

add-node:
	.\tbb.exe run \
	--datadir=$(NODE_DATA_DIR) \
	--ip=$(NODE_IP) \
	--port=$(NODE_PORT) \
	--bootstrap-ip=$(BOOT_IP) \
	--bootstrap-port=$(BOOT_PORT) \
	--disable-ssl \
	--miner=$(NODE_ADDRESS)

new-wallet:
	.\tbb.exe wallet new-account --datadir=$(BOOT_DATA_DIR)
