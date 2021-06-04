FROM alpine:3.13.5

# Create some directories and text files
RUN mkdir /rootdir && \
	(echo 'file1.txt initial contents' > /rootdir/file1.txt) && \
	(echo 'file2.txt initial contents' > /rootdir/file2.txt) && \
	mkdir /rootdir/subdir1 && \
	(echo 'sfile1.txt initial contents' > /rootdir/subdir1/sfile1.txt) && \
	(echo 'sfile2.txt initial contents' > /rootdir/subdir1/sfile2.txt)

# Create some more directories and text files
RUN mkdir /rootdir/subdir2 && \
	(echo 'sfile3.txt initial contents' > /rootdir/subdir2/sfile3.txt) && \
	(echo 'sfile4.txt initial contents' > /rootdir/subdir2/sfile4.txt)

# Update some text files
RUN touch /rootdir/file1.txt && \
	(echo 'sfile1.txt updated contents' > /rootdir/subdir1/sfile1.txt)

# Remove files and directories
RUN rm /rootdir/file1.txt && \
	rm /rootdir/subdir2/sfile3.txt
RUN rm -r -f /rootdir/subdir1

# Change ownership and permissions of files and directories
RUN chmod +x /rootdir/file2.txt && \
	chown -R 1000:1000 /rootdir/subdir2
