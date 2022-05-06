# STOR(EDGE) 

## IT
### Riassunto del progetto

Il seguente progetto è un sistema di storage distribuito di tipo chiave:valore, basato sul paradigma dell'edge computing: tale paradigma prevede un'architettura di sistema per cui ci sono diversi nodi eterogenei (sensori, dispositivi mobile, micro data-center) ai bordi della rete, quindi vicini agli utenti che utilizzano l'applicazione. Essendo tali nodi edge caratterizzati da una scarsa capacità di memorizzazione, vengono integrati con i servizi Cloud moderni, in modo da migliorare la scalabilità rispetto ai dati, andando a salvare valori poco acceduti o di grandi dimensioni.

I client che si connettono al sistema possono utilizzare la seguente API, basata su RPC: 

- Put(__chiave__, __valore__): salva un valore su un nodo edge. Per semplicità, vengono considerati solo valori di tipo testuale;
- Get(__chiave__): recupera il valore associato ad una chiave;
- Append(__chiave__, __valore__): appende un valore ad una chiave già esistente;
- Del(__chiave__): cancella un record chiave:valore

### Deployment del sistema
È richiesto che vengano deployati prima i servizi di DynamoDB ed il register, per poi passare ai nodi edge. Tutti quanti i servizi cloud effettuano il deployment facendo alcune assunzioni:

- sono presenti, all'interno della macchina su cui avviene il deployment, i seguenti due file sotto _$HOME/.aws_:
	- _credentials_;
	- _config_

Una guida per scrivere tali file può essere consultata [qui](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)
È necessario inoltre avere installato la CLI di AWS, che è possibile ottenere mediante il [seguente link](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html)

#### Nodo register
Il sistema utilizza un servizio di registrazione, utile a diversi scopi: 

- per permettere ai nodi edge di collegarsi fra di loro, in modo tale da poter replicare i dati e permettere ulteriore scalabilità;
- per far si che un client riceva la lista dei nodi edge attivi e si colleghi a quello con la latenza di rete minore

Il nodo register può essere deployato nel Cloud, all'interno di una istanza EC2, nel seguente modo:

- creare una istanza di EC2 di tipo t2-micro, secondo quanto riportato [nel seguente link](https://docs.aws.amazon.com/efs/latest/ug/gs-step-one-create-ec2-resources.html);
- collegarsi all'istanza di EC2 creata mediante ssh: __ssh -i chiave\_privata\_utente ec2-user@DNS IPv4 pubblico__;
- copiare il file binario del register, ottenuto a seguito del comando __go build register_node/__, mediante scp: __scp -i _chaive\_privata\_utente_.pem register_node/register_node ec2-user@_DNS IPv4 pubblico_:~/.__;
- lanciare il register, mediante __./register_server &__, se si ha intenzione di lanciarlo in background per poter poi chiudere la connessione via ssh, altrimenti senza &.

#### Tabella di DynamoDB
Il servizio Cloud usato per salvare i dati di grandi dimensioni e scarsamente acceduti è DynamoDB. Per il deployment della tabella di DynamoDB è stato predisposto un apposito file che fa utilizzo del tool di automazione __terraform__: tale file è il _cloud\_infrastructure.tf_ all'interno della directory _tf\_files_. Viene quindi dato per scontato che nella macchina da cui viene lanciato il programma si abbia installata una versione di __terraform__.

#### Cluster di nodi server
I nodi server sono pensati per essere deployati all'interno di container docker, per poi essere orchestrati tramite __docker-compose__, in modo da testare l'applicazione sulla macchina locale. Per fare ciò, vi è lo script shell __edge_deploy.sh__, che richiede che siano installati all'interno della macchina dove avviene il deployment:

- docker
- docker-compose (v1.27 >)

All'interno dello script:

- i nodi edge si collegano al servizio di regsitrazione, quindi occorre passare l'indirizzo IP come secondo argomento nel comando __CMD__  del _Dockerfile.edge_;
- il Dockerfile copia i file locali _credentials_ e _config_  all'interno del container così che ogni nodo possa connettersi al servizio di DynamoDB;
- il cluster di nodi edge viene lanciato col comando __docker-compose up__, utilizzando la configurazione definita nel file _docker-compose.yml_ file, contenuto nella cartella _edge\_deploy/_.


## ENG

### Project summary
The project implements a key:value based distributed storage system, using edge computing paradigm: such paradigm is based on an architecture in which there are different eterogeneous(controlla) nodes (sensor, mobile nodes, micro data-center) palced on the edges of the network, so that they are closed to the clients of the application.Such edge nodes are characterized by a limited storage capacity, so they are enhanced with modern Cloud storage services, to archieve scalability, saving heavy data and not frequently accessed ones.
Clients that connect to the system can use the following API, based on RPC: 

- Put(__key__, __value__): save a key:value on an edge node. For simplicity, the values are textual;
- Get(__key__): retrieve the value associated to a specific key;
- Append(__key__, __value__): append a value to an existing key;
- Del(__key__): delete a key:value record.

### System deployment 
To make things work, it is required first that DynamoDB and the register are deployed and then edge nodes can be deployed. Each service deployed on Cloud uses the following assumptions: 

- on the deployment machine, there are these file under the _$HOME/.aws_:
	- _credentials_;
	- _config_

A guide usefull to learn how to write these files can be found [here](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html).
A version of the AWS CLI needs to be installed, it can be done by following [this link](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html)

#### Register node
The system uses a register service to achieve multiple objectives:

- allow edge nodes to connect each other, so that data replication can sucessfully happen to archieve scalability;
- allow a client to receive the list of the active edge nodes, in order to connect to the one with the lowest network latency.

The register service is implemented ad hoc,the code is in the _register\_node_ folder, and it can be deployed in Cloud by doing the following:

- create an EC2 instance, type t2-micro is ok, following [this guide](https://docs.aws.amazon.com/efs/latest/ug/gs-step-one-create-ec2-resources.html);;
- connect to the EC2 instance created using ssh: __ssh -i user\_private\_key.pem ec2-user@DNS public IPv4__;
- copy the register binary, obtained by launching __go build register_node/__, on EC2 using scp: __scp -i _user\_private\_key_.pem register_node/register_node ec2-user@_DNS public IPv4_:~/.__;
- run the register, using __./register_server &__,if you want to run it in background so that it is possible to close the ssh connection, or without &.

#### DynamoDB table
The Cloud service used to store heavy or not frequently accessed data is DynamoDB. To deploy the DynamoDB table, there is a file that uses the automation tool __terraform__ : such file is _tf\_files/cloud\_infrastructure.tf_.So, it is necessary to have installed on the machine used to depploy the system a version of __terraform__, after this the table can be managed by doing the following: in the _tf\_files_ directory, from a shell:

- run __terraform init__ to initialize terraform backend;
- run __terraform apply__ ro create the table; 

#### Cluster di nodi server
Server nodes are thought to be deployed into Docker containers. In this case, to test that the applicatino work correctly without a suited edge computing platform, one can use __docker-compose__. To do so, there is the shell script __edge_deploy.sh__, so it is needed that the deploying machine has the following software installed:

- docker
- docker-compose (v1.27 >)

Furthermore, insde the script:

- edge nodes will connect to the register service, so the public IP address (in case of deployment on AWS EC2) needs to be set in the _Dockerfile.edge_, as second argument for the __CMD__ command; 
- the Dockerfile copies the local files _credentials_ and _config_ into the container, so that each edge node can connect to the DynamoDB service;
- the edge nodes cluster is launched by using the command __docker-compose up__, using the configuration defined in the _docker-compose.yml_ file, under the  _edge\_deploy/_ directory;

