进入java-sdk-demo中，使用"./gradlew build"编译项目即可使用。
如果遇到GoogleJavaFormat报错，可以使用"./gradlew GoogleJavaFormat"即可解决。
进入 dist目录下，使用
java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceDpkiAcme validation 10000 10000 1
测试
dist目录下有一键测试脚本dpkiacme.sh，dacme.sh，dcv.sh。

需要添加合约的java版本就要把代码添加到java-sdk-demo\src\main\java\org\fisco\bcos\sdk\demo\contract中，新压测程序添加到java-sdk-demo\src\main\java\org\fisco\bcos\sdk\demo\perf中即可。