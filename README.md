# 精简版docker

- 开发环境:ubuntu14.04 ; 内核版本 : 3.13
- 同步环境: 在ubuntu中进行开发，使用git+github保持mac端的代码同步

## 配置环境中遇到的问题

1. 在ubuntu中安装GO语言环境，在下载软件包的过程中注意不要在官网上下载，速度会很慢，又被墙的风险
2. 在vscode中搭建go开发环境时，如果使用vscode自带的go插件下载工具，会下载失败。因为vscode直接从go官方网站下载这些插件，官方网站被屏蔽了会无法下载。正确的做法是从github上下载对应的工具包，让后在相应的目录下安装即可。
3. 在ubuntu中初步使用git+github时，需要将git生成的本地密钥添加到github中，同时还要给本机设置一个用户名和邮箱，在每次上传文件到github时会要去输入github的账号和密码，需要将个人的账号信息添加到相应的文件中，即可解决这个问题。
4. 从github上下载的代码包之间的应用关系很容器出问题，因为你自己项目所建立的包目录结构有很大的出入，需要手动调整一下包的目录接口，才能使引用关系正确。


## 支持的指令

### 可交互 -ti
```
mydocker run -ti ls
mydocker run -ti -v /root/from1:/to1 
mydocker run -ti -m 100m  stress --vm-bytes 200m --vm-keep -m 1  // 限制内存100m
mydocker run -ti -cpuset 
mydocker run -ti -cpushare 512 stress --vm-bytes 200m --vm-keep -m 1 // 限制cpushare 

```


### 后台执行 -d

- 以后台运行的方式从镜像busybox启动容器container1 ,并将容器内部的数据卷挂载在宿主机/root/from1目录中,执行top指令

```
mydocker run -d --name container1 -v /root/from1:/to1 busybox top  
```

- 查看指定容器的日志

```
mydocker logs containerName
```

- 查看正在运行容器的状态

```
mydcoker ps 
```

- 进入到处于后台运行状态的容器,对容器进行操作

```
mydocker exec containerNmae sh
```

- 终止容器名为contaierName容器的运行

```
mydocker stop containerName 

```

- 删除处于stop状态的容器

```
mydocker rm containerName
```

## 开发过程

### 2018年11月30日

成功将书本第二章的内容实现，完成了最基本的运行，使用了namespace创造了基本的隔离环境。整体运行良好。

### 2018年12月2日

将cgroups功能添加入系统当中，但是在测试的时候总是无法完成隔离，在测试中观察到如下的现象：

在```/sys/fs/memory```对应文件的```memory.limit_in_bytes```和```tasks```文件中，没有数据写入，依次为依据排查出是创建memory.go中的处理逻辑写错。将这点改正之后，系统正常运行。

到这里为止，成功的使用了namespace和cgroups技术创建了一个简单的容器。但是容器内的目录还是当前运行程序的目录

### 2018年12月3日

#### aufs文件系统 
mydocker增加了aufs文件系统功能，使用busybox作为最底层的基础镜像。通过mydocker run -ti sh，启动的镜像将以busybox最为基础镜像，具体aufs文件系统的实现如下：

- 在启动容器时，通过解压busybox.tar到文件夹/root/busybox/中来作为基础镜像是只读层，同时创建/root/writeLayer/文件夹作为镜像的读写层，创建/root/mnt作为busybox和writeLayer的挂载点，通过命令```mount -t aufs -o /root/writeLayer:/root/busybox none /root/mnt```，将busybox和writeLayer挂载到mnt文件夹中。在实际的使用过程中，对容器镜像文件的修改只会体现在writeLayer中，对busybox无影响。
- 在容器退出时，会自动删除writeLayer和container-initLayer删除,在我的实现中就是将mnt和writeLayer文件夹删除。

#### 挂载volume
上述工作实现了容器和镜像的分离，缺少数据的持久保存.目前实现了数据的持久化存储，通过挂载数据卷的形式，将宿主机指定的文件夹挂载到容器内部制定的文件夹中。
实现指令如下：
```sudo ./mydocker run -ti -v /root/volume:/containerVolume ```
将宿主机/root/volume文件夹挂载到容器mnt/containerVolume中

#### 打包镜像

commit.go用来打包镜像

#### 容器后台运行

使用```mydocker run -d top```可以让top在后台运行,不过这个时候会有个bug,之前的代码在容器结束的时候会执行删除相应的writeLayer的操作,进程top在后台运行,mydocket退出后也尝试将top的writeLayer层给删除,会报```{"level":"error","msg":"Remove dir /root/writeLayer/ error remove /root/writeLayer/: device or resource busy","time":"2018-12-03T05:31:57-08:00"}```错误.


### 2018年12月4日

#### 增加ps命令
当容器在后台运行时,可以使用```mydocker ps```查看容器的运行状态

#### 增加logs命令
通过```mydocker logs containerNmae``` 查看具体容器的日志信息

#### 增加exec命令
通过```mydocker exec containerName  ```可以进入到在后台运行的容器中,对其进行操作
## 待修复的BUG

* [] 在是容器在后台运行时,mydocker尝试去删除writeLayer层.正确逻辑mydocker不会去尝试删除
* [] 错误日志不清晰,需要整理
* [] 当容器在后台运行时,如果使用kill PID的方式关闭这个容器进程,由于其/var/run/docker/containerName 文件夹的存在,依旧可以使用mydocker ps查看到容器的信息,实际上这个容器已经不存在


### 2018年12月5日

#### 存在问题

之前一直尝试在容器的/to文件夹中写入文件,想在WriteLayer中看到修改的文件,但是一直看不到,在容器其他文件中写入文件后都可以在writeLayer中看到修改信息.
容器中的/to文件夹挂载到了宿主机/root/from文件中,在/to中修改的文件都可以在from中看到,但是在WriteLayer中却看不到.

这个bug需要修复

**这不是BUG**是我理解错了,将/to挂载到宿主机的/root/from中,这是往/to中写入文件,实际上是在向/root/from中写文件,相当于一个map一样,在WriteLayer看不到/to中数据的任何改变,但是在/from中可以查看到改变.这就是挂载外部数据卷的意义.
#### 容器拥有独立隔离的文件系统

#### 增加从镜像启动容器

``` mydocker run -d --name container -v /root/from1:/to1 busybox top ```

#### 将容器打包成镜像

```mydocker run commit container image``` 即可将容器container打包成镜像image.

```mydocker run -d --name container2 -v /root/from2:/to2 image top``` 通过镜像image启动容器container2


### 2018年12月6日

#### 增加启动时传入环境变量功能 -e

- 通过一下命令在后台启动容器

``` mydocker run -d --name container -e chenxu=good -e luck=chenxu busybox top```

- 通过下面命令查看在运行状态容器的相关信息
 
 ```mydocker ps ```

- 通过下面命令进入名为container的容器中,查看环境变量

``` mydocker exec container sh ``` 可以发现启动时传入的环境变量在容器内部生效了.

**到目前为止,基本上实现了单机版本的容器,可以管理容器从启动到删除的整个生命周期,并且多个容器可以并存,使用相同的基础镜像且存储内容互不干扰**