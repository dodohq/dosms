FROM alpine:3.6
ADD dosms.out dosms.out
ENV PORT 80
EXPOSE 80
CMD ["/dosms.out"]
