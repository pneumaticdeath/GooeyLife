#!/bin/bash -e 
 
for os in darwin windows linux
do
	fyne-cross ${os} --arch=*
done

mkdir -p fyne-cross/packages || true

for dir in fyne-cross/dist/darwin-*
do
	pushd ${dir}
	zipfile="GooeyLife_$(basename ${dir} | sed -e 's/-/_/').app.zip"
	zip -r ${zipfile} GooeyLife.app
	mv ${zipfile} ../../packages/.
	popd
done

for dir in fyne-cross/dist/windows-*
do
	pushd ${dir}
	cp GooeyLife.exe.zip ../../packages/GooeyLife_$(basename ${dir} | sed -e 's/-/_/').exe.zip
	popd
done

for dir in fyne-cross/dist/linux-*
do
	pushd ${dir}
	cp GooeyLife.tar.xz ../../packages/GooeyLife_$(basename ${dir} | sed -e 's/-/_/').tar.xz
	popd
done

cat linux_block.toml >> FyneApp.toml
