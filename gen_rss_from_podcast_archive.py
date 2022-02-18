# pip install podgen
# https://podgen.readthedocs.io/en/stable/user/basic_usage_guide/part_1.html

from podgen import Podcast, Episode, Media

from glob import glob
import datetime
import os

# p.episodes.append(my_episode)
# p.rss_file('rss.xml', minimize=True, encoding = 'utf8')
def gen_pod(name, description, website):
    p = Podcast()
    p.name = name
    p.description = description
    p.website = website+'\\'
    p.explicit = False
    p.language = 'zh'
    # p.feed_url = 'http://example.com/feed.rss'
    # p.image = "https://via.placeholder.com/100"
    return p

def gen_episodes(title, link):
    my_episode = Episode()
    my_episode.title = title
    my_episode.media = retry(Media.create_from_server_response, 10, link)
    my_episode.publication_date = datetime.datetime.now(datetime.timezone.utc)
    
    return my_episode

def complete_rss(www_root, path_base, show_title):

    for pod_path in glob(f"{path_base}/{show_title}"):
    pod_dir = pod_path.split('/')[-1]
    print(pod_dir)

    p = gen_pod(pod_dir, "自制源" , www_root+f'/{pod_dir}')
    for episode_path in episode_path_list:
        path_parts = episode_path.split('/')

        episode_title = episode_path[episode_path.rfind('/')+1:episode_path.rfind('.')]
        my_episode =  gen_episodes(episode_title, 
                                   '/'.join([ www_root, path_parts[-2], path_parts[-1] ])
                                   )
        p.episodes.append(my_episode)
    p.rss_file(f"{path_base}/{pod_dir}.xml", minimize=True, encoding = "UTF-8")


def upload_rss(path_base):
    for xml in glob(f"{path_base}/"+"*.xml"):
    print(xml.split('/')[-1])
    os.system(f'''
cat "{xml}" | curl -F 'tpaste=<-' https://tpaste.us/
    ''')

www_root = "https://example.com"
path_base = "./archived"
show_titles = ["podcast_name", "podcast_name2"]


for show_title in show_titles:
    complete_rss(www_root, path_base, show_title)
upload_rss(path_base)
