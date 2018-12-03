# 重头完整的写一个精简版的docker

- 开发环境:ubuntu14.04 ; 内核版本 : 3.13
- 同步环境: 在ubuntu中进行开发，使用git+github保持mac端的代码同步

## 配置环境中遇到的问题

1. 在ubuntu中安装GO语言环境，在下载软件包的过程中注意不要在官网上下载，速度会很慢，又被墙的风险
2. 在vscode中搭建go开发环境时，如果使用vscode自带的go插件下载工具，会下载失败。因为vscode直接从go官方网站下载这些插件，官方网站被屏蔽了会无法下载。正确的做法是从github上下载对应的工具包，让后在相应的目录下安装即可。
3. 在ubuntu中初步使用git+github时，需要将git生成的本地密钥添加到github中，同时还要给本机设置一个用户名和邮箱，在每次上传文件到github时会要去输入github的账号和密码，需要将个人的账号信息添加到相应的文件中，即可解决这个问题。
4. 从github上下载的代码包之间的应用关系很容器出问题，因为你自己项目所建立的包目录结构有很大的出入，需要手动调整一下包的目录接口，才能使引用关系正确。

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