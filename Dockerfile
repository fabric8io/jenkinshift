FROM centos:7

ENV PATH $PATH:/usr/local/jenkinshift/

ADD ./bin/jenkinshift /usr/local/jenkinshift/

CMD jenkinshift
