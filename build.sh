#!/bin/bash -e 
 
for os in darwin windows linux
do
	fyne-cross ${os} --arch=*
done

fyne-cross android

# Hack to deal with a bug in the android cross 
# compile container.
if ! grep -q 'LinuxAndBSD' FyneApp.toml ; then
	cat linux-block.toml >> FyneApp.toml
fi

fyne package --target web

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
