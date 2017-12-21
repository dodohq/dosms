FROM alpine:3.6
ADD dosms.out dosms.out
EXPOSE 80
CMD ["/dosms.out"]
