FROM alpine:3.13.5

RUN mkdir /rootdir && \
	touch /rootdir/file1.txt && \
	touch /rootdir/file2.txt && \
	mkdir /rootdir/subdir1 && \
	touch /rootdir/subdir1/sfile1.txt && \
	touch /rootdir/subdir1/sfile2.txt

RUN mkdir /rootdir/subdir2 && \
	touch /rootdir/subdir2/sfile3.txt && \
	touch /rootdir/subdir2/sfile4.txt

RUN rm /rootdir/file1.txt
RUN rm /rootdir/subdir2/sfile3.txt
RUN rm -r -f /rootdir/subdir1
